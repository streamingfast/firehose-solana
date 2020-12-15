package solanadb

import (
	"context"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
)

type DB interface {
	DBReader
	DBWriter

	Close() error
}

type DBReader interface {
}

type DBWriter interface {
	// this is used to bootstrap the loader pipeline
	GetLastWrittenIrreversibleSlotRef(ctx context.Context) (ref bstream.BlockRef, err error)

	PutSlot(ctx context.Context, slot *pbcodec.Slot) error
	UpdateNowIrreversibleSlot(ctx context.Context, blk *pbcodec.Block) error
	// Flush MUST be called or you WILL lose data
	Flush(context.Context) error
}
