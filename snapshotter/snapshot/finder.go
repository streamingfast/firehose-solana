package snapshot

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"time"

	"cloud.google.com/go/storage"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Finder struct {
	*shutter.Shutter
	bucket            string
	prefix            string
	workdir           string
	snapshotPrefix    string //"sol-mainnet/snapshots"
	destinationBucket string
}

func NewFinder(bucket string, prefix string, destinationBucket string, snapshotPrefix string, workdir string) *Finder {
	finder := &Finder{
		Shutter:           shutter.New(),
		bucket:            bucket,
		prefix:            prefix,
		workdir:           workdir,
		snapshotPrefix:    snapshotPrefix,
		destinationBucket: destinationBucket,
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
	zlog.Info("Launching", zap.String("bucket", f.bucket), zap.String("prefix", f.prefix))
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}

	var validSnapshot = regexp.MustCompile(`^[0-9]*/.*$`)
	var snapshotPrefix = regexp.MustCompile(`^[0-9]*`)

	object, err := listFiles(ctx, client, f.bucket, f.prefix, nil)
	if err != nil {
		f.Shutdown(err)
	}

	uniqueSnapshots := map[string]bool{}
	for _, o := range object {
		zlog.Debug("filtering object", zap.String("object", o))
		if validSnapshot.MatchString(o) {
			zlog.Debug("found a snapshot", zap.String("object", o))
			snapshot := snapshotPrefix.FindString(o)
			uniqueSnapshots[snapshot] = true
		}
	}

	var snapshots []string
	for s, _ := range uniqueSnapshots {
		snapshots = append(snapshots, s)
	}
	sort.Strings(snapshots)

	zlog.Info("found snapshot", zap.Int("count", len(snapshots)))
	if snapshots != nil {
		snapshot := snapshots[len(snapshots)-1]
		zlog.Info("will process snapshot", zap.String("snapshot", snapshot))

		pcr := NewProcessor(f.destinationBucket, f.snapshotPrefix, f.workdir)
		err := pcr.processSnapshot(ctx, client, snapshot, f.bucket)
		if err != nil {
			f.Shutdown(err)
		}
	}

	zlog.Info("WAITING FOR 200 HOURs")
	time.Sleep(200 * time.Hour)
	//should not reach that code
	f.Shutdown(fmt.Errorf("unexpect shutdown"))
	return nil
}
