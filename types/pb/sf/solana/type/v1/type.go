package pbsol

import (
	"time"

	"github.com/streamingfast/bstream"
)

func (b *Block) Num() uint64 {
	return b.Slot
}

func (b *Block) Time() time.Time {
	if b.BlockTime == nil {
		return time.Unix(0, 0)
	}
	return time.Unix(int64(b.BlockTime.Timestamp), 0)
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
