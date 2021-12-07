package serumhist

import (
	"fmt"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
	"github.com/streamingfast/sf-solana/codec"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"github.com/streamingfast/sf-solana/serumhist/metrics"
	"go.uber.org/zap"
)

func (i *Injector) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	i.setHealthy()

	block := blk.ToNative().(*pbcodec.Block)
	forkObj := obj.(*forkable.ForkableObject)

	if forkObj.Step == forkable.StepNew {
		metrics.HeadBlockNumber.SetUint64(block.Number)
		metrics.HeadBlockTimeDrift.SetBlockTime(block.Time())
		return nil
	}

	if err := i.handler.ProcessBlock(blk, obj); err != nil {
		return err
	}

	i.slotMetrics.slotCount++

	if block.Number%logEveryXSlot == 0 {
		opts := i.slotMetrics.dump()
		opts = append(opts, []zap.Field{
			zap.Uint64("slot_number", block.Number),
			codec.ZapBase58("slot_id", block.Id),
			codec.ZapBase58("previous_id", block.PreviousId),
		}...)

		zlog.Info(fmt.Sprintf("processed %d slot", logEveryXSlot),
			opts...,
		)
	}
	return nil
}
