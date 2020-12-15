package kv

import (
	"context"
	"fmt"

	"github.com/dfuse-io/bstream"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
)

func (db *DB) Flush(ctx context.Context) error {
	return db.store.FlushPuts(ctx)
}

func (db *DB) GetLastWrittenIrreversibleSlotRef(ctx context.Context) (ref bstream.BlockRef, err error) {
	panic("implement me")
}

func (db *DB) UpdateNowIrreversibleSlot(ctx context.Context, blk *pbcodec.Block) error {
	panic("implement me")
}

func (db *DB) PutSlot(ctx context.Context, slot *pbcodec.Slot) error {
	if traceEnabled {
		db.logger.Debug("put slot")
	}

	if err := db.processSerumSlot(ctx, slot); err != nil {
		return fmt.Errorf("put slot: unable to process serum slot: %w", err)
	}

	return nil
}