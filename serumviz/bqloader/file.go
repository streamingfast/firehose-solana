package bqloader

import (
	"fmt"

	"github.com/mr-tron/base58"
)

type FileName struct {
	Prefix         string
	StartBlockNum  uint64
	LatestBlockNum uint64
	StartBlockId   []byte
	LatestBlockId  []byte
}

func NewFileName(prefix string, startBlockNum uint64, latestBlockNum uint64, startBlockId, latestBlockId []byte) *FileName {
	return &FileName{
		Prefix:         prefix,
		StartBlockNum:  startBlockNum,
		LatestBlockNum: latestBlockNum,
		StartBlockId:   startBlockId,
		LatestBlockId:  latestBlockId,
	}
}

func (f *FileName) String() string {
	return fmt.Sprintf("%s/%d-%d-%s-%s",
		f.Prefix,
		f.StartBlockNum,
		f.LatestBlockNum,
		base58.Encode(f.StartBlockId),
		base58.Encode(f.LatestBlockId),
	)
}
