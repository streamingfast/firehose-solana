package kvloader

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist"

	"go.uber.org/zap"
)

func (kv *KVLoader) processSerumNewOrders(events []*serumhist.NewOrder) interface{} {
	for _, event := range events {
		zlog.Debug("serum new order",
			zap.Stringer("side", event.Order.Side),
			zap.Stringer("market", event.Market),
			zap.Stringer("trader", event.Trader),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
			zap.String("trx_hash", event.TrxHash),
		)

		if err := kv.writeNewOrder(event); err != nil {
			return fmt.Errorf("unable to write fill event: %w", err)
		}
	}
	return nil

}

func (kv *KVLoader) processSerumFills(events []*serumhist.FillEvent) error {
	for _, event := range events {
		trader, err := kv.cache.getTrader(kv.ctx, event.TradingAccount)
		if err != nil {
			return fmt.Errorf("unable to retrieve trader for trading key %q: %w", event.TradingAccount.String(), err)
		}

		if trader == nil {
			zlog.Warn("unable to find trader for trading account, skipping fill",
				zap.Stringer("trading_account", event.TradingAccount),
				zap.Uint64("slot_number", event.SlotNumber),
				zap.Uint32("trx_id", event.TrxIdx),
				zap.Uint32("inst_id", event.InstIdx),
				zap.Stringer("market", event.Market),
				zap.String("trx_hash", event.TrxHash),
			)
			return nil
		}

		// we need to make sure we assign the trader before we proto encode, not all the keys contains the trader
		event.Trader = *trader
		event.Fill.Trader = trader.String()

		zlog.Debug("serum new fill",
			zap.Stringer("side", event.Fill.Side),
			zap.Stringer("market", event.Market),
			zap.Stringer("trader", trader),
			zap.Stringer("trading_Account", event.TradingAccount),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
		)

		if err = kv.writeFill(event); err != nil {
			return fmt.Errorf("unable to write fill event: %w", err)
		}

	}
	return nil
}

func (kv *KVLoader) processSerumOrdersCancelled(events []*serumhist.OrderCancelled) error {
	for _, event := range events {
		zlog.Debug("serum order cancelled",
			zap.Stringer("market", event.Market),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
			zap.Uint32("trx_idx", event.TrxIdx),
			zap.Uint32("inst_idx", event.InstIdx),
		)

		if err := kv.writeOrderCancelled(event); err != nil {
			return fmt.Errorf("unable to write order canceled event: %w", err)
		}
	}

	return nil
}

func (kv *KVLoader) processSerumOrdersClosed(events []*serumhist.OrderClosed) error {
	for _, event := range events {
		zlog.Debug("serum order closed",
			zap.Stringer("market", event.Market),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
			zap.Uint32("trx_idx", event.TrxIdx),
			zap.Uint32("inst_idx", event.InstIdx),
		)

		if err := kv.writeOrderClosed(event); err != nil {
			return fmt.Errorf("unable to write order closed event: %w", err)
		}
	}

	return nil

}

func (kv *KVLoader) processSerumOrdersExecuted(events []*serumhist.OrderExecuted) error {
	for _, event := range events {
		zlog.Debug("serum order executed",
			zap.Stringer("market", event.Market),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
			zap.Uint32("trx_idx", event.TrxIdx),
			zap.Uint32("inst_idx", event.InstIdx),
		)

		if err := kv.writeOrderExecuted(event); err != nil {
			return fmt.Errorf("unable to write order executed event: %w", err)
		}
	}
	return nil
}
