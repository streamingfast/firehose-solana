package pbsol

import (
	"time"

	"github.com/streamingfast/bstream"
)

func (c *Block) Num() uint64 {
	return c.Slot
}

func (c *Block) Time() time.Time {
	if c.BlockTime == nil {
		return time.Unix(0, 0)
	}
	return time.Unix(int64(c.BlockTime.Timestamp), 0)
}

func (b *Block) ID() string {
	return b.Blockhash
}

func (b *Block) PreviousID() string {
	return b.PreviousBlockhash
}

func (b *Block) AsRef() bstream.BlockRef {
	return bstream.NewBlockRef(b.ID(), b.Num())
}
