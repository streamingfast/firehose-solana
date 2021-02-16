package bqloader

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/dfuse-io/solana-go"
	"go.uber.org/zap"
)

func (bq *BQLoader) writeTradingAccount(tradingAccount, trader solana.PublicKey) error {
	// TODO store trading account & trader
	return nil
}

func (bq *BQLoader) processSerumFills(events []*serumhist.FillEvent) error {
	for _, event := range events {
		zlog.Debug("serum new fill",
			zap.Stringer("side", event.Fill.Side),
			zap.Stringer("market", event.Market),
			zap.Stringer("trading_Account", event.TradingAccount),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
		)

		if bq.orderFilledTable == nil {
			return nil
		}
		if err := bq.orderFilledTable.Inserter().Put(bq.ctx, event.Fill); err != nil {
			return fmt.Errorf("unable to store fills: %w", err)
		}
	}
	return nil
}

func (bq *BQLoader) processSerumNewOrders(events []*serumhist.NewOrder) error {
	for _, event := range events {
		if bq.orderCreatedTable == nil {
			return nil
		}

		row := &Row{
			mapping: bq.orderCreatedMapping,
			event:   event.Order,
		}

		if err := bq.orderCreatedTable.Inserter().Put(bq.ctx, row); err != nil {
			return fmt.Errorf("unable to store fills: %w", err)
		}
	}
	return nil
}
