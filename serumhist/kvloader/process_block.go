package kvloader

import (
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	pbserumhist "github.com/streamingfast/sf-solana/pb/sf/solana/serumhist/v1"
	"github.com/streamingfast/sf-solana/serumhist"
	"go.uber.org/zap"
)

func (kv *KVLoader) ProcessBlock(blk *bstream.Block, obj interface{}) error {

	block := blk.ToNative().(*pbcodec.Block)
	forkObj := obj.(*forkable.ForkableObject)

	// this flow will eventually change to process the list of
	// proto meta objects
	serumSlot := forkObj.Obj.(*serumhist.SerumSlot)
	for _, ta := range serumSlot.TradingAccountCache {
		err := kv.cache.setTradingAccount(kv.ctx, ta.TradingAccount, ta.Trader)
		if err != nil {
			return fmt.Errorf("unable to store trading account %d (%s): %w", block.Number, base58.Encode(block.Id), err)
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
		LastWrittenBlockNum: block.Number,
		LastWrittenBlockId:  block.Id,
	}); err != nil {
		return fmt.Errorf("error while saving block checkpoint: %w", err)
	}

	t := block.Time()
	if err := kv.flushIfNeeded(block.Number, t); err != nil {
		zlog.Error("flushIfNeeded", zap.Error(err))
		return err
	}
	return nil
}
