package bqloader

import (
	"context"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
)

func (bq *BQLoader) setCheckpoints(ctx context.Context) error {
	for _, table := range []string{newOrder, fillOrder, tradingAccount} {
		tableCheckpoint, err := bq.loader.ReadCheckpoint(ctx, table)
		if err != nil {
			return err
		}
		bq.checkpoints[table] = tableCheckpoint
	}
	return nil
}

func (bq *BQLoader) GetCheckpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {
	if len(bq.checkpoints) == 0 {
		err := bq.setCheckpoints(ctx)
		if err != nil {
			return nil, err
		}
	}

	var earliestCheckpoint *pbserumhist.Checkpoint
	for _, table := range []string{newOrder, fillOrder, tradingAccount} {
		tableCheckpoint, ok := bq.checkpoints[table]
		if !ok {
			continue
		}

		if tableCheckpoint == nil { // one or more checkpoints not set.  return nil.  caller will handle nil checkpoint
			return nil, nil
		}

		if earliestCheckpoint == nil {
			earliestCheckpoint = tableCheckpoint
			continue
		}

		if tableCheckpoint.LastWrittenSlotNum < earliestCheckpoint.LastWrittenSlotNum {
			earliestCheckpoint = tableCheckpoint
		}
	}

	return earliestCheckpoint, nil
}
