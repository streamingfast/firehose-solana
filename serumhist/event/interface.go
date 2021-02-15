package event

import (
	"time"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/solana-go"
)

type Ref struct {
	market      solana.PublicKey
	orderSeqNum uint64
	slotNumber  uint64
	trxHash     string
	trxIdx      uint32
	instIdx     uint32
	slotHash    string
	timestamp   time.Time
}

type NewOrder struct {
	Ref
	order *pbserumhist.Order
}

func (e *NewOrder) WriteTo(w Writer) error {
	return w.NewOrder(e)
}

type Fill struct {
	Ref
	tradingAccount solana.PublicKey
	fill           *pbserumhist.Fill
}

func (e *Fill) WriteTo(w Writer) error {
	return w.Fill(e)
}

type OrderExecuted struct {
	Ref
}

func (e *OrderExecuted) WriteTo(w Writer) error {
	return w.OrderExecuted(e)
}

type OrderClosed struct {
	Ref
	instrRef *pbserumhist.InstructionRef
}

func (e *OrderClosed) WriteTo(w Writer) error {
	return w.OrderClosed(e)
}

type OrderCancelled struct {
	Ref
	instrRef *pbserumhist.InstructionRef
}

func (e *OrderCancelled) WriteTo(w Writer) error {
	return w.OrderCancelled(e)
}

type Writeable interface {
	WriteTo(writer Writer) error
}

type Writer interface {
	NewOrder(*NewOrder) error
	Fill(*Fill) error
	OrderExecuted(*OrderExecuted) error
	OrderClosed(*OrderClosed) error
	OrderCancelled(*OrderCancelled) error
}
