package serumhist

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

const (
	DatabaseTimeout = 10 * time.Minute
)

func (i *Injector) writeCheckpoint(ctx context.Context, slot *pbcodec.Slot) error {

	key := keyer.EncodeCheckpoint()

	checkpoint := &pbserumhist.Checkpoint{
		LastWrittenSlotNum: slot.Number,
		LastWrittenSlotId:  slot.Id,
	}

	value, err := proto.Marshal(checkpoint)
	if err != nil {
		return err
	}

	return i.kvdb.Put(ctx, key, value)
}

func (i *Injector) GetShardCheckpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {

	key := keyer.EncodeCheckpoint()

	ctx, cancel := context.WithTimeout(ctx, DatabaseTimeout)
	defer cancel()

	val, err := i.kvdb.Get(ctx, key)
	if err == store.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error while reading checkpoint: %w", err)
	}

	// Decode val as `pbaccounthist.ShardCheckpoint`
	out := &pbserumhist.Checkpoint{}
	if err := proto.Unmarshal(val, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (i *Injector) resolveCheckpoint(ctx context.Context, startBlockNum uint64, ignoreCheckpointOnLaunch bool) (*pbserumhist.Checkpoint, error) {
	if ignoreCheckpointOnLaunch {
		checkpoint := newCheckpoint(startBlockNum)
		zlog.Info("ignoring checkpoint on launch starting without a checkpoint",
			zap.Reflect("checkpoint", checkpoint),
		)
		return checkpoint, nil
	}

	// Retrieved lastProcessedBlock must be in the shard's range, and that shouldn't
	// change across invocations, or in the lifetime of the database.
	checkpoint, err := i.GetShardCheckpoint(ctx)
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
