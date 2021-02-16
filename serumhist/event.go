package serumhist

import (
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

type FillEvent struct {
	*Ref
	TradingAccount solana.PublicKey
	Trader         solana.PublicKey
	Fill           *pbserumhist.Fill
}

type OrderExecuted struct {
	*Ref
}

type OrderClosed struct {
	*Ref
	InstrRef *pbserumhist.InstructionRef
}

type OrderCancelled struct {
	*Ref
	InstrRef *pbserumhist.InstructionRef
}
