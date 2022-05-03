// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.19.4
// source: confirmed_block.proto

package pbsolana

import (
	"github.com/streamingfast/bstream"
)

func (c *ConfirmedBlock) Num() uint64 {
	if c.BlockHeight != nil {
		return c.BlockHeight.GetBlockHeight()
	}
	return 0
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