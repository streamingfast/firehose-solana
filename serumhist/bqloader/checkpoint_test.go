package bqloader

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/dfuse-io/dstore"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

func Test_getLatestCheckpointFromFiles(t *testing.T) {
	var fileNames []string
	prefix := "testPrefix"
	timestamp := "2020-01-01-12345"
	fileNames = append(fileNames, "testPrefix/invalid_file_should_not_affect_outcome")
	for _, fi := range []struct {
		startSlot    uint64
		latestSlot   uint64
		startSlotId  string
		latestSlotId string
	}{
		{
			startSlot:    0,
			latestSlot:   100,
			startSlotId:  "a",
			latestSlotId: "b",
		},
		{
			startSlot:    101,
			latestSlot:   200,
			startSlotId:  "c",
			latestSlotId: "d",
		},
		{
			startSlot:    201,
			latestSlot:   300,
			startSlotId:  "e",
			latestSlotId: "f",
		},
	} {
		fileNames = append(fileNames, NewFileName(prefix, fi.startSlot, fi.latestSlot, fi.startSlotId, fi.latestSlotId, timestamp).String())
	}
	store := getMockStore(fileNames)

	checkpoint, err := getLatestCheckpointFromFiles(context.Background(), store, prefix)

	require.NoError(t, err)
	assert.Equal(t, checkpoint.LastWrittenSlotNum, uint64(300))
	assert.Equal(t, checkpoint.LastWrittenSlotId, "f")
}

func getMockStore(files []string) dstore.Store {
	store := dstore.NewMockStore(func(base string, f io.Reader) (err error) {
		return nil
	})
	for i, file := range files {
		store.SetFile(file, []byte(fmt.Sprintf("%d", i)))
	}
	return store
}
