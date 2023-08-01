package pbsol

import (
	"time"

	"github.com/mr-tron/base58"
)

func (b *Block) GetFirehoseBlockID() string {
	return base58.Encode(b.Id)
}

func (b *Block) GetFirehoseBlockNumber() uint64 {
	return b.Number
}

func (b *Block) GetFirehoseBlockParentID() string {
	return base58.Encode(b.PreviousId)
}

func (b *Block) GetFirehoseBlockTime() time.Time {
	return time.Unix(int64(b.GenesisUnixTimestamp), 0)
}

func (x *Block) GetFirehoseBlockLIBNum() uint64 {
	return 0
}
