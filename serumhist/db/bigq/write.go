package bigq

import (
	"context"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	serumhistdb "github.com/dfuse-io/dfuse-solana/serumhist/db"
)

func (b *Bigq) NewOrder(ctx context.Context, order *serumhistdb.NewOrder) error {
	if b.orderCreatedTable == nil {
		return nil
	}

	row := &Row{
		mapping: b.orderCreatedMapping,
		event:   order,
	}
	return b.orderCreatedTable.Inserter().Put(ctx, row)

}

func (b *Bigq) Fill(ctx context.Context, fill *serumhistdb.Fill) error {
	if b.orderFilledTable == nil {
		return nil
	}
	return b.orderFilledTable.Inserter().Put(ctx, fill)

}

func (b *Bigq) OrderExecuted(ctx context.Context, executed *serumhistdb.OrderExecuted) error {
	return nil
}

func (b *Bigq) OrderClosed(ctx context.Context, closed *serumhistdb.OrderClosed) error {
	return nil
}

func (b *Bigq) OrderCancelled(ctx context.Context, cancelled *serumhistdb.OrderCancelled) error {
	return nil
}

func (b *Bigq) WriteCheckpoint(ctx context.Context, checkpoint *pbserumhist.Checkpoint) error {
	panic("implement me")
}

func (b *Bigq) Flush(ctx context.Context) (err error) {
	panic("implement me")
}
