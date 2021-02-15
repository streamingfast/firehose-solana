package db

import (
	"context"
	"time"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/solana-go"
)

type Ref struct {
	Market      solana.PublicKey
	OrderSeqNum uint64
	SlotNumber  uint64
	TrxHash     string
	TrxIdx      uint32
	InstIdx     uint32
	SlotHash    string
	Timestamp   time.Time
}

func (r *Ref) GetEventRef() *Ref {
	return r
}

type NewOrder struct {
	*Ref
	Order *pbserumhist.Order
}

func (e *NewOrder) WriteTo(ctx context.Context, w Writer) error {
	return w.NewOrder(ctx, e)
}

type Fill struct {
	*Ref
	TradingAccount solana.PublicKey
	Trader         solana.PublicKey
	Fill           *pbserumhist.Fill
}

func (e *Fill) WriteTo(ctx context.Context, w Writer) error {
	return w.Fill(ctx, e)
}

type OrderExecuted struct {
	*Ref
}

func (e *OrderExecuted) WriteTo(ctx context.Context, w Writer) error {
	return w.OrderExecuted(ctx, e)
}

type OrderClosed struct {
	*Ref
	InstrRef *pbserumhist.InstructionRef
}

func (e *OrderClosed) WriteTo(ctx context.Context, w Writer) error {
	return w.OrderClosed(ctx, e)
}

type OrderCancelled struct {
	*Ref
	InstrRef *pbserumhist.InstructionRef
}

func (e *OrderCancelled) WriteTo(ctx context.Context, w Writer) error {
	return w.OrderCancelled(ctx, e)
}
