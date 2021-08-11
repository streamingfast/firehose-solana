package serumhist

import (
	"context"
	"fmt"

	"github.com/streamingfast/bstream"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"go.uber.org/zap"
)

func (i *Injector) resolveCheckpoint(ctx context.Context, startBlockNum uint64, ignoreCheckpointOnLaunch bool) (*pbserumhist.Checkpoint, error) {
	if ignoreCheckpointOnLaunch {
		checkpoint := newCheckpoint(startBlockNum)
		zlog.Info("ignoring checkpoint on launch starting without a checkpoint",
			zap.Reflect("checkpoint", checkpoint),
		)
		return checkpoint, nil
	}

	zlog.Info("retrieving serumhist checkpoint from handler")
	// Retrieved lastProcessedBlock must be in the shard's range, and that shouldn't
	// change across invocations, or in the lifetime of the database.
	checkpoint, err := i.checkpointResolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching checkpoint: %w", err)
	}

	if checkpoint != nil {
		zlog.Info("found checkpoint",
			zap.Reflect("checkpoint", checkpoint),
		)
		return checkpoint, nil
	}

	checkpoint = newCheckpoint(startBlockNum)
	zlog.Info("starting without checkpoint",
		zap.Reflect("checkpoint", checkpoint),
	)
	return checkpoint, nil
}

func newCheckpoint(startBlock uint64) *pbserumhist.Checkpoint {
	if startBlock <= bstream.GetProtocolFirstStreamableBlock {
		startBlock = bstream.GetProtocolFirstStreamableBlock
	}
	return &pbserumhist.Checkpoint{
		LastWrittenSlotNum: startBlock,
	}
}
