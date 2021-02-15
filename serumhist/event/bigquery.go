package event

import (
	"context"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
)

type BigQuery struct {
}

func (b *BigQuery) NewOrder(ctx context.Context, order *NewOrder) error {
	panic("implement me")
}

func (b *BigQuery) Fill(ctx context.Context, fill *Fill) error {
	panic("implement me")
}

func (b *BigQuery) OrderExecuted(ctx context.Context, executed *OrderExecuted) error {
	panic("implement me")
}

func (b *BigQuery) OrderClosed(ctx context.Context, closed *OrderClosed) error {
	panic("implement me")
}

func (b *BigQuery) OrderCancelled(ctx context.Context, cancelled *OrderCancelled) error {
	panic("implement me")
}

func (b *BigQuery) WriteCheckpoint(ctx context.Context, slot *pbcodec.Slot) error {
	panic("implement me")
}

func (b *BigQuery) Checkpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {
	panic("implement me")
}

func (b *BigQuery) Flush(ctx context.Context) (err error) {
	panic("implement me")
}
