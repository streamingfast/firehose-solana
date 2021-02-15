package bqloader

import (
	"fmt"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"go.uber.org/zap"
)

func (i *BQLoader) ProcessBlock(blk *bstream.Block, obj interface{}) error {

	slot := blk.ToNative().(*pbcodec.Slot)
	forkObj := obj.(*forkable.ForkableObject)

	serumSlot := forkObj.Obj.(*serumSlot)

	for _, ta := range serumSlot.tradingAccountCache {
		err := i.cache.setTradingAccount(i.ctx, ta.tradingAccount, ta.trader)
		if err != nil {
			return fmt.Errorf("unable to store trading account %d (%s): %w", slot.Number, slot.Id, err)
		}
	}

	// process close
	i.slotMetrics.serumFillCount += len(serumSlot.orderFilledEvents)

	if err := i.db.Write(serumSlot); err {

	}

	//if err := i.processSerumFills(serumSlot.orderFilledEvents); err != nil {
	//	return fmt.Errorf("unable to process serum order orderFilledEvents: %w", err)
	//}
	//
	//if err := i.processSerumOrdersExecuted(serumSlot.orderExecutedEvents); err != nil {
	//	return fmt.Errorf("unable to process serum orders executed: %w", err)
	//}
	//
	//if err := i.processSerumOrdersCancelled(serumSlot.orderCancelledEvents); err != nil {
	//	return fmt.Errorf("unable to process serum orders cancelled: %w", err)
	//}
	//
	//if err := i.processSerumOrdersClosed(serumSlot.orderClosedEvents); err != nil {
	//	return fmt.Errorf("unable to process serum orders executed: %w", err)
	//}

	if err := i.db.WriteCheckpoint(i.ctx, &pbserumhist.Checkpoint{
		LastWrittenSlotNum: slot.Number,
		LastWrittenSlotId:  slot.Id,
	}); err != nil {
		return fmt.Errorf("error while saving block checkpoint: %w", err)
	}

	if err := i.flush(i.ctx, slot); err != nil {
		return fmt.Errorf("error while flushing: %w", err)
	}

	t := slot.Block.Time()

	if err := i.flushIfNeeded(slot.Number, t); err != nil {
		zlog.Error("flushIfNeeded", zap.Error(err))
		return err
	}

	i.slotMetrics.slotCount++

	if slot.Number%logEveryXSlot == 0 {
		opts := i.slotMetrics.dump()
		opts = append(opts, []zap.Field{
			zap.Uint64("slot_number", slot.Number),
			zap.String("slot_id", slot.Id),
			zap.String("previous_id", slot.PreviousId),
			zap.Int("trading_account_cached_count", len(serumSlot.tradingAccountCache)),
			zap.Int("fill_count", len(serumSlot.orderFilledEvents)),
		}...)

		zlog.Info(fmt.Sprintf("processed %d slot", logEveryXSlot),
			opts...,
		)
	}
	return nil
}
