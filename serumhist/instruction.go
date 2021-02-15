package serumhist

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist/db"
	"go.uber.org/zap"
)

func (i Injector) processSerumFills(events []*db.Fill) error {
	for _, event := range events {
		trader, err := i.cache.getTrader(i.ctx, event.TradingAccount)
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

		// push the events to subscription
		i.manager.emit(event)

		if err = f.WriteFill(i.ctx, event); err != nil {
			return fmt.Errorf("unable to write fill event: %w", err)
		}

	}
	return nil
}

func (i *Injector) processSerumOrdersCancelled(events []*db.OrderCancelled) error {
	for _, event := range events {
		zlog.Debug("serum order cancelled",
			zap.Stringer("market", event.Market),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
			zap.Uint32("trx_idx", event.TrxIdx),
			zap.Uint32("inst_idx", event.InstIdx),
		)

		if i.manager != nil {
			// push the events to subscription
			i.manager.emit(event)
		}

		if err := i.db.OrderCancelled(i.ctx, event); err != nil {
			return fmt.Errorf("unable to write order canceled event: %w", err)
		}
	}

	return nil
}

func (i *Injector) processSerumOrdersClosed(events []*db.OrderClosed) error {
	for _, event := range events {
		zlog.Debug("serum order closed",
			zap.Stringer("market", event.Market),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
			zap.Uint32("trx_idx", event.TrxIdx),
			zap.Uint32("inst_idx", event.InstIdx),
		)

		if i.manager != nil {
			// push the events to subscription
			i.manager.emit(event)
		}

		if err := i.db.OrderClosed(i.ctx, event); err != nil {
			return fmt.Errorf("unable to write order closed event: %w", err)
		}
	}

	return nil

}

func (i *Injector) processSerumOrdersExecuted(events []*db.OrderExecuted) error {
	for _, event := range events {
		zlog.Debug("serum order executed",
			zap.Stringer("market", event.Market),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
			zap.Uint32("trx_idx", event.TrxIdx),
			zap.Uint32("inst_idx", event.InstIdx),
		)

		if i.manager != nil {
			// push the events to subscription
			i.manager.emit(event)
		}

		if err := i.db.OrderExecuted(i.ctx, event); err != nil {
			return fmt.Errorf("unable to write order executed event: %w", err)
		}
	}
	return nil
}
