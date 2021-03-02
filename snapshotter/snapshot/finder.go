package snapshot

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Finder struct {
	*shutter.Shutter
	sourceBucket               string
	sourceSnapshotsPrefix      string
	workdir                    string
	destinationSnapshotsFolder string //"sol-mainnet/snapshots"
	destinationBucket          string
}

func NewFinder(sourceBucket string, sourceSnapshotsPrefix string, destinationBucket string, destinationSnapshotsFolder string, workdir string) *Finder {
	finder := &Finder{
		Shutter:                    shutter.New(),
		sourceBucket:               sourceBucket,
		sourceSnapshotsPrefix:      sourceSnapshotsPrefix,
		destinationBucket:          destinationBucket,
		destinationSnapshotsFolder: destinationSnapshotsFolder,
		workdir:                    workdir,
	}

	return finder
}

func (f *Finder) Launch() error {
	go func() {
		err := f.launch()
		if err != nil {
			f.Shutdown(err)
		}
	}()

	return nil
}

func (f *Finder) launch() error {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	zlog.Info("Launching", zap.String("sourceBucket", f.sourceBucket), zap.String("sourceSnapshotsPrefix", f.sourceSnapshotsPrefix))
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}

	var validSnapshot = regexp.MustCompile(`^[0-9]*/.*$`)
	var snapshotPrefix = regexp.MustCompile(`^[0-9]*`)

	for c := ticker; ; <-c.C {
		select {
		case <-f.Terminating():
			return nil
		default:
		}

		object, err := listFiles(ctx, client, f.sourceBucket, f.sourceSnapshotsPrefix, nil)
		if err != nil {
			return err
		}

		uniqueSnapshots := map[int64]bool{}
		for _, o := range object {
			zlog.Debug("filtering object", zap.String("object", o))
			if validSnapshot.MatchString(o) {
				zlog.Debug("found a snapshot", zap.String("object", o))
				snapshot := snapshotPrefix.FindString(o)

				slot, err := strconv.ParseInt(snapshot, 10, 64)
				if err != nil {
					f.Shutdown(err)
				}
				uniqueSnapshots[slot] = true
			}
		}

		var snapshots []int64
		for s, _ := range uniqueSnapshots {
			snapshots = append(snapshots, s)
		}
		sort.Slice(snapshots, func(i, j int) bool {
			return snapshots[i] > snapshots[j]
		})

		zlog.Info("found snapshot", zap.Int("count", len(snapshots)))
		if snapshots != nil {
			sourceSnapshotName := snapshots[0]
			zlog.Info("will process sourceSnapshotName", zap.Int64("sourceSnapshotName", sourceSnapshotName))

			pcr := NewProcessor(f.sourceBucket, fmt.Sprintf("%d", sourceSnapshotName), f.destinationBucket, f.destinationSnapshotsFolder, f.workdir, client)
			completed, err := pcr.CompletedSnapshot(ctx)
			if err != nil {
				return fmt.Errorf("error checking if snapshot was already processed: %w", err)
			}

			if completed {
				zlog.Info("snapshot already processed. skipping", zap.Int64("snapshot", sourceSnapshotName))
				continue
			}

			err = pcr.processSnapshot(ctx)
			if err != nil {
				return err
			}
		}
	}
}
