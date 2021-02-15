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

type Fill struct {
	Ref
	tradingAccount solana.PublicKey
	fill           *pbserumhist.Fill
}

type OrderExecuted struct {
	Ref
}

type OrderClosed struct {
	Ref
	instrRef *pbserumhist.InstructionRef
}

type OrderCancelled struct {
	Ref
	instrRef *pbserumhist.InstructionRef
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
