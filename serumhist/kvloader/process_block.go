package kvloader

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"go.uber.org/zap"
)

func (kv *KVLoader) ProcessBlock(blk *bstream.Block, obj interface{}) error {

	slot := blk.ToNative().(*pbcodec.Slot)
	forkObj := obj.(*forkable.ForkableObject)

	// this flow will eventually change to process the list of
	// proto meta objects
	serumSlot := forkObj.Obj.(*serumhist.SerumSlot)
	for _, ta := range serumSlot.TradingAccountCache {
		err := kv.cache.setTradingAccount(kv.ctx, ta.TradingAccount, ta.Trader)
		if err != nil {
			return fmt.Errorf("unable to store trading account %d (%s): %w", slot.Number, slot.Id, err)
		}
	}

	if err := kv.processSerumNewOrders(serumSlot.OrderNewEvents); err != nil {
		return fmt.Errorf("unable to process serum new order: %w", err)
	}

	if err := kv.processSerumFills(serumSlot.OrderFilledEvents); err != nil {
		return fmt.Errorf("unable to process serum order fill events: %w", err)
	}

	if err := kv.processSerumOrdersExecuted(serumSlot.OrderExecutedEvents); err != nil {
		return fmt.Errorf("unable to process serum orders executed: %w", err)
	}

	if err := kv.processSerumOrdersCancelled(serumSlot.OrderCancelledEvents); err != nil {
		return fmt.Errorf("unable to process serum orders cancelled: %w", err)
	}

	if err := kv.processSerumOrdersClosed(serumSlot.OrderClosedEvents); err != nil {
		return fmt.Errorf("unable to process serum orders executed: %w", err)
	}

	if err := kv.writeCheckpoint(&pbserumhist.Checkpoint{
		LastWrittenSlotNum: slot.Number,
		LastWrittenSlotId:  slot.Id,
	}); err != nil {
		return fmt.Errorf("error while saving block checkpoint: %w", err)
	}

	t := slot.Block.Time()
	if err := kv.flushIfNeeded(slot.Number, t); err != nil {
		zlog.Error("flushIfNeeded", zap.Error(err))
		return err
	}
	return nil
}
