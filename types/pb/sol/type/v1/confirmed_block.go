package pbsolana

import (
	"time"

	"github.com/streamingfast/bstream"
)

func (c *ConfirmedBlock) Num() uint64 {
	if c.BlockHeight != nil {
		return c.BlockHeight.GetBlockHeight()
	}
	return 0
}

func (c *ConfirmedBlock) Time() time.Time {
	if c.BlockTime == nil {
		return time.Unix(0, 0)
	}
	return time.Unix(int64(c.BlockTime.Timestamp), 0)
}

func (b *ConfirmedBlock) ID() string {
	return b.Blockhash
}

func (b *ConfirmedBlock) PreviousID() string {
	return b.PreviousBlockhash
}

func (b *ConfirmedBlock) AsRef() bstream.BlockRef {
	return bstream.NewBlockRef(b.ID(), b.Num())
}
