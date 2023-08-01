package pbsol

import (
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/mr-tron/base58"
)

func (x *Block) GetFirehoseBlockID() string {
	return x.Blockhash
}

func (x *Block) GetFirehoseBlockNumber() uint64 {
	return x.Slot
}

func (x *Block) GetFirehoseBlockParentID() string {
	return x.PreviousBlockhash
}

func (x *Block) GetFirehoseBlockTime() time.Time {
	if x.BlockTime == nil {
		return time.Unix(0, 0)
	}
	return time.Unix(int64(x.BlockTime.Timestamp), 0)
}

func (x *Block) GetFirehoseBlockLIBNum() uint64 {
	return x.ParentSlot
}

func (x *Block) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("num", x.Slot)
	encoder.AddString("id", x.Blockhash)
	return nil
}

func (x *ConfirmedTransaction) AsBase58String() string {
	return base58.Encode(x.Transaction.Signatures[0])
}
