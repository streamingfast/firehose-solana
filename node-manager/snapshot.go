package nodemanager

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/dfuse-io/dstore"
	"go.uber.org/zap"
)

func (s *Superviser) TakeSnapshot(snapshotStore dstore.Store, numberOfSnapshotsToKeep int) error {
	ctx := context.Background()
	files, err := ioutil.ReadDir(s.localSnapshotDir)
	for _, f := range files {
		snapshotName := f.Name()
		if !f.IsDir() &&
			strings.HasPrefix(snapshotName, "snapshot-") &&
			strings.HasSuffix(snapshotName, ".tar.zst") {
			zlog.Info("found snapshot file", zap.String("snapshot_dir", s.localSnapshotDir), zap.String("snapshot_name", snapshotName))

			exist, err := snapshotStore.FileExists(ctx, snapshotName)
			if err != nil {
				return fmt.Errorf("checking snapshot existance: %s: %w", snapshotName, err)
			}

			if !exist && !s.currentlyUploading(snapshotName) {
				go s.uploadSnapshot(ctx, snapshotName, snapshotStore)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("walking snapshot dir: %s: %w", s.localSnapshotDir, err)
	}

	// todo; purge holder snap shot
	// If there are snapshot that are older than `numberOfSnapshotsToKeep`, then wipe them
	return nil
}

func (s *Superviser) uploadSnapshot(ctx context.Context, snapshotName string, snapshotStore dstore.Store) {
	s.uploadingJobs[snapshotName] = new(interface{})
	defer delete(s.uploadingJobs, snapshotName)

	snapshotPath := path.Join(s.localSnapshotDir, snapshotName)
	snapshotFile, err := os.Open(snapshotPath)
	if err != nil {
		zlog.Error("failed opening snapshot file, will retry on next TakeSnapshot call", zap.String("snapshot_path", snapshotPath))
		return
	}

	uploadCtx, cancel := context.WithTimeout(ctx, 1*time.Hour)
	defer cancel()

	err = snapshotStore.WriteObject(uploadCtx, snapshotName, snapshotFile)
	if err != nil {
		zlog.Error("failed snapshot upload, will retry on next TakeSnapshot call", zap.String("snapshot_dir", s.localSnapshotDir), zap.String("snapshot_name", snapshotName))
		return
	}
}

func (s *Superviser) currentlyUploading(snapshotName string) bool {
	_, ok := s.uploadingJobs[snapshotName]
	return ok
}

func (s *Superviser) RestoreSnapshot(snapshotName string, snapshotStore dstore.Store) error {
	if snapshotName == "latest" {
		// find latest
	}

	if snapshotName == "before-last-merged" {

		// 63090809
		// 63590809
		// 64090809
		//   merged at  64090800
		// 64092834
		//  merged at  64092800 ?
		//  merged at  64099900 ?

		// find the snapshot the CLOSEST before the LAST MERGED BLOCK FILES we can find.
	}

	// otherwise, use that as the block number ?!

	// take that snapshot, download it
	// remove the other snapshots from the local directory..
	return nil
}
