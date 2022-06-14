package nodemanager

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/streamingfast/dstore"
	"go.uber.org/zap"
)

type Snapshotter struct {
	localDir         string
	genesisURL       string
	snapshotStore    dstore.Store
	mergedBlockStore dstore.Store
	uploadingJobs    map[string]interface{}
}

func NewSnapshotter(localDir, genesisURL string, snapshotStore, mergedBlockStore dstore.Store) *Snapshotter {
	return &Snapshotter{
		localDir:         localDir,
		genesisURL:       genesisURL,
		snapshotStore:    snapshotStore,
		mergedBlockStore: mergedBlockStore,
		uploadingJobs:    map[string]interface{}{},
	}
}

func (s *Snapshotter) Restore(snapshotName string) error {
	ctx := context.Background()
	var mergedFileSlots []uint64
	snapshotsVsMergeFile := map[uint64]string{}
	if snapshotName == "latest" {
		panic("latest not implemented yet")
	}

	if snapshotName == "before-last-merged" {
		zlog.Info("walking snapshot folder before last merger", zap.String("snapshot_name", snapshotName))
		err := s.snapshotStore.Walk(ctx, "", func(filename string) (err error) {
			zlog.Info("found snapshot", zap.String("file_name", filename))
			//snapshot-64506076-2qzVcbpcSwhxqtD7wjwgvAHWriSZtfJtynWQ4syS43mb.tar.zst
			//           64506000
			parts := strings.Split(filename, "-")
			slot, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return fmt.Errorf("parse slot to int: %s: %w", parts[0], err)
			}
			mergeFileBeforeSnapshot := uint64(slot/100) * 100
			mergeFileAfterSnapshot := mergeFileBeforeSnapshot + 100
			mergeSlotNum := mergeFileAfterSnapshot
			mergedFileSlots = append(mergedFileSlots, mergeSlotNum)
			snapshotsVsMergeFile[mergeSlotNum] = filename
			return nil
		})

		if err != nil {
			return fmt.Errorf("walking snapshots: %w", err)
		}

		sort.Slice(mergedFileSlots, func(i, j int) bool { return mergedFileSlots[i] > mergedFileSlots[j] })
		found := false
		for _, mergedSlot := range mergedFileSlots {
			zlog.Info("looking for merge file", zap.Uint64("merged_slot", mergedSlot))
			mergeFileName := fmt.Sprintf("%010d", mergedSlot)
			exists, err := s.mergedBlockStore.FileExists(ctx, mergeFileName)
			if err != nil {
				return fmt.Errorf("merger file exists: %s: %w", mergeFileName, err)
			}

			if exists {
				snapshotName := snapshotsVsMergeFile[mergedSlot]
				zlog.Info("found snapshot", zap.String("snapshot_name", snapshotName), zap.Uint64("merged_slot", mergedSlot))
				if err := s.restoreFrom(ctx, snapshotName, s.snapshotStore); err != nil {
					return fmt.Errorf("restoring from: %s: %w", snapshotName, err)
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("failed to find a snapshot")
		}
	}

	return nil
}

func (s *Snapshotter) RequiresStop() bool {
	// TODO this is a guess, needs to be tested
	return true
}

func (s *Snapshotter) Backup(lastSeenBlockNum uint32) (string, error) {
	ctx := context.Background()
	files, err := ioutil.ReadDir(s.localDir)
	for _, f := range files {
		snapshotName := f.Name()
		if !f.IsDir() &&
			strings.HasPrefix(snapshotName, "snapshot-") &&
			strings.HasSuffix(snapshotName, ".tar.zst") {
			zlog.Info("found snapshot file", zap.String("snapshot_dir", s.localDir), zap.String("snapshot_name", snapshotName))

			exist, err := s.snapshotStore.FileExists(ctx, snapshotName)
			if err != nil {
				return "", fmt.Errorf("checking snapshot existance: %s: %w", snapshotName, err)
			}

			if !exist && !s.currentlyUploading(snapshotName) {
				go s.uploadSnapshot(ctx, snapshotName, s.snapshotStore)
			}
		}
	}

	if err != nil {
		return "", fmt.Errorf("walking snapshot dir: %s: %w", s.localDir, err)
	}

	// TODO: this is a weird behavior trying to match the interface
	// built in node-manager
	return "latest", nil
}

func (s *Snapshotter) currentlyUploading(snapshotName string) bool {
	_, ok := s.uploadingJobs[snapshotName]
	return ok
}

func (s *Snapshotter) uploadSnapshot(ctx context.Context, snapshotName string, snapshotStore dstore.Store) {
	s.uploadingJobs[snapshotName] = new(interface{})
	defer delete(s.uploadingJobs, snapshotName)

	snapshotPath := path.Join(s.localDir, snapshotName)
	snapshotFile, err := os.Open(snapshotPath)
	if err != nil {
		zlog.Error("failed opening snapshot file, will retry on next TakeSnapshot call", zap.String("snapshot_path", snapshotPath))
		return
	}

	uploadCtx, cancel := context.WithTimeout(ctx, 1*time.Hour)
	defer cancel()

	err = snapshotStore.WriteObject(uploadCtx, snapshotName, snapshotFile)
	if err != nil {
		zlog.Error("failed snapshot upload, will retry on next TakeSnapshot call", zap.String("snapshot_dir", s.localDir), zap.String("snapshot_name", snapshotName))
		return
	}
}

func (s *Snapshotter) List(params map[string]string) ([]string, error) {
	// TODO this is a guess
	return []string{
		"latest",
		"before-last-merged",
	}, nil
}

func (s *Snapshotter) restoreFrom(ctx context.Context, snapshotName string, snapshotStore dstore.Store) error {
	zlog.Info("restoring", zap.String("snapshot_name", snapshotName), zap.Stringer("from_store", snapshotStore.BaseURL()))
	dataFolder := s.localDir
	err := s.cleanupDataFolder(dataFolder)
	if err != nil {
		return fmt.Errorf("cleaning up folder: %s: %w", dataFolder, err)
	}

	localURL := "file://" + s.localDir
	localStore, err := dstore.NewSimpleStore(localURL)
	if err != nil {
		return fmt.Errorf("creating local snapshotStore: %s: %w", localURL, err)
	}

	snapshotReader, err := snapshotStore.OpenObject(ctx, snapshotName)
	if err != nil {
		return fmt.Errorf("open object:%s: %w", snapshotName, err)
	}

	zlog.Info("copying snapshot", zap.String("snapshot_name", snapshotName), zap.Stringer("from_store", snapshotStore.BaseURL()), zap.Stringer("to_local_store", localStore.BaseURL()))
	if err := localStore.WriteObject(ctx, snapshotName, snapshotReader); err != nil {
		return fmt.Errorf("writing snapshot:%s to local snapshotStore: %s: %w", snapshotName, localURL, err)
	}

	if err := s.copyGenesis(ctx, localStore); err != nil {
		return fmt.Errorf("copying genesis: %w", err)
	}

	dir, err := ioutil.ReadDir(s.localDir)
	if err != nil {
		return fmt.Errorf("reading data folder:%s: %w", dir, err)
	}

	for _, d := range dir {
		content := path.Join([]string{s.localDir, d.Name()}...)
		zlog.Info("element:", zap.String("content", content))
	}

	return nil
}

func (s *Snapshotter) copyGenesis(ctx context.Context, localStore dstore.Store) error {
	genesisStore, genesisFileName, err := dstore.NewStoreFromURL(s.genesisURL)
	if err != nil {
		return fmt.Errorf("creating genesis snapshotStore:%s : %w", s.genesisURL, err)
	}

	genesisReader, err := genesisStore.OpenObject(ctx, genesisFileName)
	zlog.Info("copying genesis", zap.String("file_name", genesisFileName), zap.Stringer("from_store", genesisStore.BaseURL()), zap.Stringer("to_local_store", localStore.BaseURL()))
	if err := localStore.WriteObject(ctx, genesisFileName, genesisReader); err != nil {
		return fmt.Errorf("writing genesis: %s from:%s to local snapshotStore: %s: %w", genesisFileName, genesisStore.BaseURL(), localStore.BaseURL(), err)
	}

	return nil
}

func (s *Snapshotter) cleanupDataFolder(folder string) error {
	zlog.Info("Cleaning up data folder", zap.String("folder", folder))
	dir, err := ioutil.ReadDir(folder)
	if err != nil {
		return fmt.Errorf("reading folder:%s: %w", folder, err)
	}
	for _, d := range dir {
		content := path.Join([]string{folder, d.Name()}...)
		zlog.Info("deleting content", zap.String("content", content))
		err := os.RemoveAll(content)
		if err != nil {
			zlog.Warn("failed to delete", zap.String("content", content))
		}
	}
	return nil
}
