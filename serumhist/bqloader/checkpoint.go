package bqloader

import (
	"context"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"sync"
	"time"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
)

func (bq *BQLoader) GetCheckpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {
	timeout := 120 * time.Second /// ???
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(len(bq.avroHandlers))

	checkpointChan := make(chan *pbserumhist.Checkpoint)
	go func() {
		wg.Wait()
		close(checkpointChan)
	}()

	for _, v := range bq.avroHandlers {
		go func(handler *avroHandler) {
			defer wg.Done()

			store := handler.Store
			prefix := handler.Prefix

			var highestSlotNum uint64
			var highestSlotId string
			foundAny := false

			err := store.Walk(ctx, prefix, ".tmp", func(filename string) error {
				filenameParts := strings.Split(filename, "-")
				if len(filenameParts) < 5 {
					zlog.Warn("could not parse slot num for file. skipping unknown file", zap.String("filename", filename))
					return nil
				}

				fileLatestSlotNum, err := strconv.ParseUint(filenameParts[1], 10, 64)
				if err != nil {
					zlog.Warn("could not parse slot num for file. skipping unknown file", zap.String("filename", filename))
					return nil
				}
				fileLatestSlotId := filenameParts[3]

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

			if err == context.DeadlineExceeded {
				zlog.Info("context deadline exceeded when walking store files")
				err = nil
			}

			if err != nil || !foundAny {
				zlog.Warn("could not determine checkpoint")
				return
			}

			handler.CheckpointSlotNum = highestSlotNum
			checkpointChan <- &pbserumhist.Checkpoint{
				LastWrittenSlotNum: highestSlotNum,
				LastWrittenSlotId:  highestSlotId,
			}
		}(v)
	}

	var earliestCheckpoint *pbserumhist.Checkpoint
	for checkpoint := range checkpointChan {
		if earliestCheckpoint == nil {
			earliestCheckpoint = checkpoint
			continue
		}

		if checkpoint.LastWrittenSlotNum < earliestCheckpoint.LastWrittenSlotNum {
			earliestCheckpoint = checkpoint
		}
	}

	// return lowest of the checkpoints
	return earliestCheckpoint, nil
}
