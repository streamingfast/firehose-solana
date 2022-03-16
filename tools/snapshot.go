package tools

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "",
}

var listSnapshotCmd = &cobra.Command{
	Use: "list {snapshot_bucket}",
	Short: `Prints ordered snapshot list for a given bucket. Buckets can be:
- gs://mainnet-beta-ledger-us-ny5
- gs://mainnet-beta-ledger-asia-sg1
- gs://mainnet-beta-ledger-europe-fr2
`,
	Args: cobra.ExactArgs(1),
	RunE: listSnapshotE,
}

var boundsARegEx = regexp.MustCompile(`^Ledger has data for (\d+) slots (\d+) to (\d+)`)
var boundsBRegEx = regexp.MustCompile(`^Ledger has data for slots (\d+) to (\d+)`)

func init() {
	Cmd.AddCommand(snapshotCmd)
	snapshotCmd.AddCommand(listSnapshotCmd)
	listSnapshotCmd.Flags().String("output", "snapshots.jsonl", "output filename either .csv or .jsonl")

}

func listSnapshotE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	snapshotBucket := args[0]
	zlog.Info("fetching solana snapshot", zap.String("snapshot_path", snapshotBucket))

	outputFile := viper.GetString("output")
	outputCSV := false
	if strings.HasSuffix(outputFile, ".csv") {
		outputCSV = true
	} else if !strings.HasSuffix(outputFile, ".jsonl") {
		return fmt.Errorf("expected --output to either be a .csv or .jsonl file (i.e snapshost.csv or snapshots.jsonl")
	}

	reader, err := NewSnapshotReader(snapshotBucket)
	if err != nil {
		return err
	}

	snapshots, err := reader.ReadSnapshots(ctx)
	if err != nil {
		return err
	}

	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].Slot < snapshots[j].Slot
	})

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to open a file: %w", err)
	}
	defer file.Close()

	var csvwriter *csv.Writer
	if outputCSV {
		csvwriter = csv.NewWriter(file)
		csvwriter.Write([]string{"slot", "bucket", "has bounds", "slot count", "start slot", "end slot", "version", "rocksdb path", "rocksdb compression", "snapshot path"})
		defer csvwriter.Flush()
	}

	for _, snapshot := range snapshots {
		if outputCSV {
			err = csvwriter.Write([]string{
				fmt.Sprintf("%d", snapshot.Slot),
				snapshot.Bucket,
				fmt.Sprintf("%t", snapshot.HasBound),
				fmt.Sprintf("%d", snapshot.SlotCount),
				fmt.Sprintf("%d", snapshot.StartSlot),
				fmt.Sprintf("%d", snapshot.EndSlot),
				snapshot.Version,
				snapshot.RocksDBPath,
				fmt.Sprintf("%t", snapshot.RocksDBCompressed),
				snapshot.SnapshotPath,
			})
			if err != nil {
				return fmt.Errorf("failed to write to csv: %w", err)
			}
			continue
		}
		cnt, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		file.Write(cnt)
		file.Write([]byte{'\n'})
	}

	return nil
}

type SnapshotReader struct {
	ctx    context.Context
	bucket *url.URL
	client *storage.Client
}

func NewSnapshotReader(bucket string) (*SnapshotReader, error) {
	base, err := url.Parse(bucket)
	if err != nil {
		return nil, err
	}

	client, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}

	return &SnapshotReader{
		bucket: base,
		client: client,
	}, err
}

func (sr *SnapshotReader) ReadSnapshots(ctx context.Context) ([]*Snapshot, error) {
	zlog.Info("retrieving slots")
	epochs := []*Snapshot{}
	it := sr.client.Bucket(sr.bucket.Host).Objects(ctx, &storage.Query{
		Delimiter: "/",
		Prefix:    strings.TrimLeft(sr.bucket.Path, "/"),
	})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		epoch, err := strconv.ParseUint(strings.Trim(attrs.Prefix, "/"), 10, 64)
		if err != nil {
			zlog.Debug("failed to parse epoch", zap.Error(err), zap.String("epoch", attrs.Prefix))
		}
		if epoch <= 0 {
			continue
		}

		snap, err := sr.readSnapshot(ctx, epoch)
		if err != nil {
			return nil, err
		}
		epochs = append(epochs, snap)
	}
	return epochs, nil

}

func (sr *SnapshotReader) readSnapshot(ctx context.Context, epoch uint64) (*Snapshot, error) {
	s := &Snapshot{Slot: epoch, Bucket: sr.bucket.Host}
	var snapshotRegEx = regexp.MustCompile(fmt.Sprintf("snapshot-%d-.*", epoch))
	zlog.Debug("retrieving slot detail", zap.Uint64("slot", epoch))
	ut := sr.client.Bucket(sr.bucket.Host).Objects(ctx, &storage.Query{
		Delimiter: "",
		Prefix:    fmt.Sprintf("%d/", epoch),
	})
	seenRocksDbFolder := false
	hourlySnapshot := ""
	for {
		subAttrs, err := ut.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		if strings.Contains(subAttrs.Name, "hourly") {
			if snapshotRegEx.MatchString(subAttrs.Name) {
				hourlySnapshot = fmt.Sprintf("%s/%s", strings.TrimRight(sr.bucket.String(), "/"), subAttrs.Name)
			}
			continue
		}

		if strings.Contains(subAttrs.Name, fmt.Sprintf("%d/rocksdb/", epoch)) {
			seenRocksDbFolder = true
			continue
		}

		if strings.Contains(subAttrs.Name, "rocksdb.tar.bz2") {
			s.RocksDBPath = fmt.Sprintf("%s/%s", strings.TrimRight(sr.bucket.String(), "/"), subAttrs.Name)
			s.RocksDBCompressed = true
			continue
		}

		if strings.Contains(subAttrs.Name, "bounds.txt") {
			s.HasBound = true
			s.SlotCount, s.StartSlot, s.EndSlot, err = sr.readSnapshotBound(ctx, epoch)
			if err != nil {
				return nil, err
			}
			continue
		}
		if strings.Contains(subAttrs.Name, "version.txt") {
			cnt, err := sr.readFile(ctx, epoch, "version.txt")
			if err != nil {
				return nil, err
			}
			s.Version = strings.Trim(string(cnt), "\n")
			continue
		}

		if snapshotRegEx.MatchString(subAttrs.Name) {
			s.SnapshotPath = fmt.Sprintf("%s/%s", strings.TrimRight(sr.bucket.String(), "/"), subAttrs.Name)
			continue
		}
	}
	if seenRocksDbFolder && s.RocksDBPath == "" {
		s.RocksDBPath = fmt.Sprintf("%s/%d/rocksdb", strings.TrimRight(sr.bucket.String(), "/"), s.Slot)
		s.RocksDBCompressed = false
	}
	if s.SnapshotPath == "" {
		s.SnapshotPath = hourlySnapshot
	}
	return s, nil
}

func (sr *SnapshotReader) readSnapshotBound(ctx context.Context, epoch uint64) (slotCount uint64, startSlot uint64, endSlot uint64, err error) {
	cnt, err := sr.readFile(ctx, epoch, "bounds.txt")
	if err != nil {
		return 0, 0, 0, err
	}
	matches := boundsARegEx.FindStringSubmatch(string(cnt))
	if len(matches) != 4 {
		matches := boundsBRegEx.FindStringSubmatch(string(cnt))
		if len(matches) != 3 {
			return 0, 0, 0, fmt.Errorf("unable to parse bound file for epoch %d: %s", epoch, string(cnt))
		}
		startSlot, err = strconv.ParseUint(matches[1], 10, 64)
		if err != nil {
			zlog.Debug("failed to parse bound.txt start slot", zap.Error(err), zap.String("match", matches[2]))
		}
		endSlot, err = strconv.ParseUint(matches[2], 10, 64)
		if err != nil {
			zlog.Debug("failed to parse bound.txt end slot", zap.Error(err), zap.String("match", matches[3]))
		}
		slotCount = endSlot - startSlot + 1
	} else {
		slotCount, err = strconv.ParseUint(matches[1], 10, 64)
		if err != nil {
			zlog.Debug("failed to parse bound.txt slot count", zap.Error(err), zap.String("match", matches[1]))
		}
		startSlot, err = strconv.ParseUint(matches[2], 10, 64)
		if err != nil {
			zlog.Debug("failed to parse bound.txt start slot", zap.Error(err), zap.String("match", matches[2]))
		}
		endSlot, err = strconv.ParseUint(matches[3], 10, 64)
		if err != nil {
			zlog.Debug("failed to parse bound.txt end slot", zap.Error(err), zap.String("match", matches[3]))
		}
	}

	return slotCount, startSlot, endSlot, nil
}

func (sr *SnapshotReader) readFile(ctx context.Context, epoch uint64, filename string) ([]byte, error) {
	filepath := path.Join(strings.TrimLeft(sr.bucket.Path, "/"), fmt.Sprintf("%d", epoch), filename)
	zlog.Debug("retrieving file", zap.String("filepath", filepath))
	filereader, err := sr.client.Bucket(sr.bucket.Host).Object(filepath).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(filereader)
}

func (sr *SnapshotReader) hasFile(ctx context.Context, epoch uint64, filename string) (bool, error) {
	filepath := path.Join(strings.TrimLeft(sr.bucket.Path, "/"), fmt.Sprintf("%d", epoch), filename)
	_, err := sr.client.Bucket(sr.bucket.Host).Object(filepath).Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

type Snapshot struct {
	Slot   uint64
	Bucket string
	// bound
	HasBound  bool
	SlotCount uint64
	StartSlot uint64
	EndSlot   uint64

	//version
	Version string

	// rocksdb
	RocksDBPath       string
	RocksDBCompressed bool

	//Snapshot
	SnapshotPath string
}
