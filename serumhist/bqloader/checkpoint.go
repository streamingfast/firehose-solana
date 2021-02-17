package bqloader

import (
	"context"
	"go.uber.org/zap"
	"sort"
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

	checkpoints := make([]*pbserumhist.Checkpoint, 0, len(bq.avroHandlers))
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

	for checkpoint := range checkpointChan {
		checkpoints = append(checkpoints, checkpoint)
	}

	if len(checkpoints) == 0 {
		return nil, nil
	}

	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].LastWrittenSlotNum < checkpoints[j].LastWrittenSlotNum
	})

	// return lowest of the checkpoints
	return checkpoints[0], nil
}
