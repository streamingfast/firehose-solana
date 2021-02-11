package nodemanager

import (
	"context"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dfuse-io/dstore"

	"github.com/stretchr/testify/assert"
)

//func TestSuperviser_RestoreSnapshot(t *testing.T) {
//	type fields struct {
//		Superviser        *superviser.Superviser
//		name              string
//		mergedBlocksStore dstore.Store
//		options           *Options
//		client            *rpc.Client
//		logger            *zap.Logger
//		localSnapshotDir  string
//		uploadingJobs     map[string]interface{}
//	}
//	type args struct {
//		snapshotName  string
//		snapshotStore dstore.Store
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &Superviser{
//				Superviser:        tt.fields.Superviser,
//				name:              tt.fields.name,
//				mergedBlocksStore: tt.fields.mergedBlocksStore,
//				options:           tt.fields.options,
//				client:            tt.fields.client,
//				logger:            tt.fields.logger,
//				localSnapshotDir:  tt.fields.localSnapshotDir,
//				uploadingJobs:     tt.fields.uploadingJobs,
//			}
//			if err := s.RestoreSnapshot(tt.args.snapshotName, tt.args.snapshotStore); (err != nil) != tt.wantErr {
//				t.Errorf("RestoreSnapshot() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

func TestSuperviser_TakeSnapshot(t *testing.T) {
	s := &Superviser{
		localSnapshotDir: "/tmp",
		uploadingJobs:    map[string]interface{}{},
	}

	done := make(chan bool, 1)
	snapshotName := "snapshot-123.tar.zst"
	snapshotPath := path.Join("/tmp", snapshotName)
	touchTestFile(t, snapshotPath)
	defer deleteTestFile(t, snapshotPath)

	store := dstore.NewMockStore(func(base string, f io.Reader) (err error) {
		assert.Equal(t, true, s.currentlyUploading(snapshotName))
		done <- true
		return nil
	})

	err := s.TakeSnapshot(store, 0)
	require.NoError(t, err)

	select {
	case <-time.Tick(time.Second):
		t.Error("time out")
	case r := <-done:
		assert.Equal(t, true, r)
		assert.Equal(t, false, s.currentlyUploading(snapshotName))
	}

}

func TestSuperviser_currentlyUploading(t *testing.T) {
	tests := []struct {
		uploadingJobs     map[string]interface{}
		snapshotNamp      string
		expectedUploading bool
		name              string
	}{
		{
			name:              "not uploading",
			uploadingJobs:     map[string]interface{}{},
			snapshotNamp:      "snap.1",
			expectedUploading: false,
		},
		{
			name: "uploading",
			uploadingJobs: map[string]interface{}{
				"snap.1": new(interface{}),
			},
			snapshotNamp:      "snap.1",
			expectedUploading: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := Superviser{
				uploadingJobs: test.uploadingJobs,
			}
			assert.Equal(t, test.expectedUploading, s.currentlyUploading(test.snapshotNamp))
		})
	}
}

func TestSuperviser_uploadSnapshot(t *testing.T) {
	s := &Superviser{
		localSnapshotDir: "/tmp",
		uploadingJobs:    map[string]interface{}{},
	}

	done := make(chan bool, 1)
	snapshotName := "snapshot-123.tar.zst"
	snapshotPath := path.Join("/tmp", snapshotName)
	touchTestFile(t, snapshotPath)
	defer deleteTestFile(t, snapshotPath)

	store := dstore.NewMockStore(func(base string, f io.Reader) (err error) {
		assert.Equal(t, true, s.currentlyUploading(snapshotName))
		done <- true
		return nil
	})

	s.uploadSnapshot(context.Background(), snapshotName, store)

	select {
	case <-time.Tick(time.Second):
		t.Error("time out")
	case r := <-done:
		assert.Equal(t, true, r)
		assert.Equal(t, false, s.currentlyUploading(snapshotName))
	}
}

func touchTestFile(t *testing.T, filePath string) {
	t.Helper()
	file, err := os.Create(filePath)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
}

func deleteTestFile(t *testing.T, filePath string) {
	t.Helper()
	err := os.Remove(filePath)
	if err != nil {
		t.Error(err)
	}
}
