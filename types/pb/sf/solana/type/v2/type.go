package pbsol

import (
	"time"

	"github.com/mr-tron/base58"
	"github.com/streamingfast/bstream"
)

func (b *Block) ID() string {
	return base58.Encode(b.Id)
}

func (b *Block) Num() uint64 {
	return b.Number
}

func (b *Block) PreviousID() string {
	return base58.Encode(b.PreviousId)
}

func (b *Block) Time() time.Time {
	return time.Unix(int64(b.GenesisUnixTimestamp), 0)
}

func (b *Block) AsRef() bstream.BlockRef {
	return bstream.NewBlockRef(b.ID(), b.Number)
}
