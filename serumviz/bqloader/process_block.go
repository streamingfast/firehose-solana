package bqloader

import (
	"fmt"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
	"github.com/streamingfast/sf-solana/serumhist"
)

func (bq *BQLoader) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	forkObj := obj.(*forkable.ForkableObject)

	// this flow will eventually change to process the list of proto meta objects
	serumSlot := forkObj.Obj.(*serumhist.SerumSlot)

	tradingAccountsHandler := bq.eventHandlers[tableTraders]
	for _, ta := range serumSlot.TradingAccountCache {
		_, found := bq.traderAccountCache.getTrader(ta.TradingAccount.String())
		if found {
			continue
		}

		err := bq.traderAccountCache.setTradingAccount(ta.TradingAccount.String(), ta.Trader.String())
		if err != nil {
			return fmt.Errorf("could not write trader to cache: %w", err)
		}

		account := &serumhist.TradingAccount{
			Trader:     ta.Trader,
			Account:    ta.TradingAccount,
			SlotNumber: blk.Number,
		}

		err = tradingAccountsHandler.HandleEvent(AsEncoder(account), blk.Number, blk.Id)
		if err != nil {
			return fmt.Errorf("unable to process trading account: %w", err)
		}
	}

	newOrdersEventsHandler := bq.eventHandlers[tableOrders]
	for _, e := range serumSlot.OrderNewEvents {
		err := newOrdersEventsHandler.HandleEvent(AsEncoder(e), e.SlotNumber, e.SlotHash)
		if err != nil {
			return fmt.Errorf("unable to process new order: %w", err)
		}
	}

	fillsHandler := bq.eventHandlers[tableFills]
	for _, e := range serumSlot.OrderFilledEvents {
		err := fillsHandler.HandleEvent(AsEncoder(e), e.SlotNumber, e.SlotHash)
		if err != nil {
			return fmt.Errorf("unable to process new order: %w", err)
		}
	}

	for handlerId, handler := range bq.eventHandlers {
		if err := handler.Flush(bq.ctx); err != nil {
			return fmt.Errorf("error flushing handler %q: %w", handlerId, err)
		}
	}

	return nil
}
