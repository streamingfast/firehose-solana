package bqloader

import (
	"fmt"
)

type FileName struct {
	Prefix        string
	StartSlotNum  uint64
	LatestSlotNum uint64
	StartSlotId   string
	LatestSlotId  string
}

func NewFileName(prefix string, startSlotNum uint64, latestSlotNum uint64, startSlotId, latestSlotId string) *FileName {
	return &FileName{
		Prefix:        prefix,
		StartSlotNum:  startSlotNum,
		LatestSlotNum: latestSlotNum,
		StartSlotId:   startSlotId,
		LatestSlotId:  latestSlotId,
	}
}

func (f *FileName) String() string {
	return fmt.Sprintf("%s/%d-%d-%s-%s",
		f.Prefix,
		f.StartSlotNum,
		f.LatestSlotNum,
		f.StartSlotId,
		f.LatestSlotId,
	)
}
