package bqloader

import (
	"fmt"
	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/dfuse-io/dfuse-solana/serumhist/db"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"go.uber.org/zap"
)

func (bq *BQLoader) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	slot := blk.ToNative().(*pbcodec.Slot)
	forkObj := obj.(*forkable.ForkableObject)

	serumSlot := forkObj.Obj.(*serumhist.SerumSlot)

	if err := i.db.Write(serumSlot); err {

	}

	for _, e := range serumSlot.Events {
		switch v := e.(type) {
		}
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

	if err := bq.db.WriteCheckpoint(i.ctx, &pbserumhist.Checkpoint{
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

	return nil
}
