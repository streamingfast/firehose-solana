package db

import (
	"context"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
)

type Eventeable interface {
	GetEventRef() *Ref
}

type Writeable interface {
	WriteTo(writer Writer) error
}

type DB interface {
	Writer
	SerumReader
	StatsReader
	Close()
}

type Writer interface {
	Write(slot *SerumSlot) error

	NewOrder(context.Context, *NewOrder) error
	Fill(context.Context, *Fill) error
	OrderExecuted(context.Context, *OrderExecuted) error
	OrderClosed(context.Context, *OrderClosed) error
	OrderCancelled(context.Context, *OrderCancelled) error

	WriteCheckpoint(ctx context.Context, checkpoint *pbserumhist.Checkpoint) error
	GetCheckpoint(ctx context.Context) (*pbserumhist.Checkpoint, error)
	Flush(ctx context.Context) (err error)
}

type SerumReader interface {
}

type StatsReader interface {
}
