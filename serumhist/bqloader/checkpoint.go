package bqloader

import (
	"context"
	"sync"
	"time"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dstore"
	"go.uber.org/zap"
)

func (bq *BQLoader) GetCheckpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {
	timeout := 120 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(len(bq.eventHandlers))

	checkpointChan := make(chan *pbserumhist.Checkpoint)
	errChan := make(chan error)
	go func() {
		wg.Wait()
		close(checkpointChan)
		close(errChan)
	}()

	for _, h := range bq.eventHandlers {
		go func(handler *eventHandler) {
			defer wg.Done()

			checkpoint, err := getLatestCheckpointFromFiles(ctx, handler.Store, handler.TableName)
			if err != nil {
				errChan <- err
				return
			}

			if checkpoint == nil {
				return
			}

			handler.SetCheckpoint(checkpoint.LastWrittenSlotNum)
			checkpointChan <- checkpoint
		}(h)
	}

	var earliestCheckpoint *pbserumhist.Checkpoint
	for checkpoint := range checkpointChan {
		if checkpoint == nil {
			continue
		}

		if earliestCheckpoint == nil {
			earliestCheckpoint = checkpoint
			continue
		}

		if checkpoint.LastWrittenSlotNum < earliestCheckpoint.LastWrittenSlotNum {
			earliestCheckpoint = checkpoint
		}
	}

	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	default:
	}

	return earliestCheckpoint, nil
}

func getLatestCheckpointFromFiles(ctx context.Context, store dstore.Store, prefix string) (checkpoint *pbserumhist.Checkpoint, err error) {
	var highestSlotNum uint64
	var highestSlotId string
	foundAny := false

	err = store.Walk(ctx, prefix, "", func(filename string) error {
		fn, err := parseLatestInfoFromFilePath(filename)
		if err != nil {
			zlog.Warn("could not parse file. skipping unknown file",
				zap.String("filename", filename),
				zap.Error(err),
			)
			return nil
		}
		fileLatestSlotNum := fn.LatestSlotNum
		fileLatestSlotId := fn.LatestSlotId

		if !foundAny {
			highestSlotNum = fileLatestSlotNum
			highestSlotId = fileLatestSlotId
			foundAny = true
			return nil
		}

		if fileLatestSlotNum <= highestSlotNum {
			return nil
		}

		highestSlotNum = fileLatestSlotNum
		highestSlotId = fileLatestSlotId
		return nil
	})

	if foundAny {
		checkpoint = &pbserumhist.Checkpoint{
			LastWrittenSlotNum: highestSlotNum,
			LastWrittenSlotId:  highestSlotId,
		}
	}

	return
}
