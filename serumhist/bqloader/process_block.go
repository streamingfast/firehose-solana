package bqloader

import (
	"context"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist"
	"go.uber.org/zap"
)

func (bq *BQLoader) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	slot := blk.ToNative().(*pbcodec.Slot)
	forkObj := obj.(*forkable.ForkableObject)

	// this flow will eventually change to process the list of proto meta objects
	serumSlot := forkObj.Obj.(*serumhist.SerumSlot)
	for _, ta := range serumSlot.TradingAccountCache {
		if err := bq.processTradingAccount(ta.TradingAccount, ta.Trader, slot.Number, slot.Id); err != nil {
			return fmt.Errorf("unable to store trading account %d (%s): %w", slot.Number, slot.Id, err)
		}
	}

	if err := bq.processSerumNewOrders(serumSlot.OrderNewEvents); err != nil {
		return fmt.Errorf("unable to process serum new orders : %w", err)
	}

	if err := bq.processSerumFills(serumSlot.OrderFilledEvents); err != nil {
		return fmt.Errorf("unable to process serum order fill events: %w", err)
	}

	var flushError error
	for _, handler := range bq.avroHandlers {
		if err := handler.FlushIfNeeded(context.Background()); err != nil {
			zlog.Error("avro flush", zap.Error(err))
			flushError = err
		}
	}
	if flushError != nil {
		return flushError
	}

	return nil
}
