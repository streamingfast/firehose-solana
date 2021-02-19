package bqloader

import (
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/dfuse-solana/serumhist"
)

func (bq *BQLoader) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	forkObj := obj.(*forkable.ForkableObject)

	// If a serumhist-firehose is doing the job, then we'll receive this:
	// blk.Meta
	// blk.Transactions[].Meta
	// blk.Transactions[].Instructions[].Meta

	for _, meta := range blk.Meta {
		bq.dispatch(meta)
	}

	for _, trx := range blk.Transactions {
		for _, meta := range trx.Meta {
			bq.dispatch(meta)
		}

		for _, inst := range trx.Instructions {
			for _, meta := range inst.Meta {
				bq.dispatch(meta)
			}
		}
	}

// this flow will eventually change to process the list of proto meta objects
	serumSlot := forkObj.Obj.(*serumhist.SerumSlot)
	for _, ta := range serumSlot.TradingAccountCache {
		if err := bq.processTradingAccount(ta.TradingAccount, ta.Trader, blk.Number, blk.Id); err != nil {
			return fmt.Errorf("unable to store trading account %d (%s): %w", blk.Number, blk.Id, err)
		}
	}

	if err := bq.processSerumNewOrders(serumSlot.OrderNewEvents); err != nil {
		return fmt.Errorf("unable to process serum new orders : %w", err)
	}

	if err := bq.processSerumFills(serumSlot.OrderFilledEvents); err != nil {
		return fmt.Errorf("unable to process serum order fill events: %w", err)
	}

	for handlerId, handler := range bq.avroHandlers {
		if err := handler.FlushIfNeeded(bq.ctx); err != nil {
			return fmt.Errorf("error flushing handler %q: %w", handlerId, err)
		}
	}

	return nil
}
