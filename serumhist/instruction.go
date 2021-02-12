package serumhist

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	kvdb "github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

func (i Injector) processSerumFills(orderFillEvents []*orderFillEvent) (out []*kvdb.KV, err error) {
	for _, orderFillEvent := range orderFillEvents {
		trader, err := i.cache.getTrader(i.ctx, orderFillEvent.tradingAccount)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve trader for trading key %q: %w", orderFillEvent.tradingAccount.String(), err)
		}

		if trader == nil {
			zlog.Warn("unable to find trader for trading account, skipping fill",
				zap.Stringer("trading_account", orderFillEvent.tradingAccount),
				zap.Uint64("slot_number", orderFillEvent.slotNumber),
				zap.Uint32("trx_id", orderFillEvent.trxIdx),
				zap.Uint32("inst_id", orderFillEvent.instIdx),
				zap.Stringer("market", orderFillEvent.market),
			)
			return nil, nil
		}

		// we need to make sure we assign the trader before we proto encode, not all the keys contains the trader
		orderFillEvent.fill.Trader = trader.String()
		cnt, err := proto.Marshal(orderFillEvent.fill)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal to fill: %w", err)
		}

		zlog.Debug("serum new fill",
			zap.Stringer("side", orderFillEvent.fill.Side),
			zap.Stringer("market", orderFillEvent.market),
			zap.Stringer("trader", trader),
			zap.Stringer("trading_Account", orderFillEvent.tradingAccount),
			zap.Uint64("order_seq_num", orderFillEvent.orderSeqNum),
			zap.Uint64("slot_num", orderFillEvent.slotNumber),
		)

		// push the events to subscription
		i.manager.emit(orderFillEvent, orderFillEvent.orderSeqNum, orderFillEvent.market)

		out = append(out, []*kvdb.KV{
			{
				Key:   keyer.EncodeFill(orderFillEvent.market, orderFillEvent.slotNumber, uint64(orderFillEvent.trxIdx), uint64(orderFillEvent.instIdx), orderFillEvent.orderSeqNum),
				Value: cnt,
			},
			{
				Key: keyer.EncodeFillByTrader(*trader, orderFillEvent.market, orderFillEvent.slotNumber, uint64(orderFillEvent.trxIdx), uint64(orderFillEvent.instIdx), orderFillEvent.orderSeqNum),
			},
			{
				Key: keyer.EncodeFillByTraderMarket(*trader, orderFillEvent.market, orderFillEvent.slotNumber, uint64(orderFillEvent.trxIdx), uint64(orderFillEvent.instIdx), orderFillEvent.orderSeqNum),
			},
		}...)
	}
	return
}

func (i *Injector) processSerumOrdersCancelled(orderCancelledEvents []*orderCancelledEvent) (out []*kvdb.KV, err error) {
	for _, orderCancelledEvent := range orderCancelledEvents {
		zlog.Debug("serum order cancelled",
			zap.Stringer("market", orderCancelledEvent.market),
			zap.Uint64("order_seq_num", orderCancelledEvent.orderSeqNum),
			zap.Uint64("slot_num", orderCancelledEvent.slotNumber),
			zap.Uint32("trx_idx", orderCancelledEvent.trxIdx),
			zap.Uint32("inst_idx", orderCancelledEvent.instIdx),
		)

		val, err := proto.Marshal(orderCancelledEvent.instrRef)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal to fill: %w", err)
		}

		// push the events to subscription
		i.manager.emit(orderCancelledEvent, orderCancelledEvent.orderSeqNum, orderCancelledEvent.market)

		out = append(out, &kvdb.KV{
			Key:   keyer.EncodeOrderCancel(orderCancelledEvent.market, orderCancelledEvent.slotNumber, uint64(orderCancelledEvent.trxIdx), uint64(orderCancelledEvent.instIdx), orderCancelledEvent.orderSeqNum),
			Value: val,
		})
	}

	return
}

func (i *Injector) processSerumOrdersClosed(orderClosedEvents []*orderClosedEvent) (out []*kvdb.KV, err error) {
	for _, orderClosedEvent := range orderClosedEvents {
		zlog.Debug("serum order closed",
			zap.Stringer("market", orderClosedEvent.market),
			zap.Uint64("order_seq_num", orderClosedEvent.orderSeqNum),
			zap.Uint64("slot_num", orderClosedEvent.slotNumber),
			zap.Uint32("trx_idx", orderClosedEvent.trxIdx),
			zap.Uint32("inst_idx", orderClosedEvent.instIdx),
		)

		val, err := proto.Marshal(orderClosedEvent.instrRef)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal to fill: %w", err)
		}

		// push the events to subscription
		i.manager.emit(orderClosedEvent, orderClosedEvent.orderSeqNum, orderClosedEvent.market)

		out = append(out, &kvdb.KV{
			Key:   keyer.EncodeOrderClose(orderClosedEvent.market, orderClosedEvent.slotNumber, uint64(orderClosedEvent.trxIdx), uint64(orderClosedEvent.instIdx), orderClosedEvent.orderSeqNum),
			Value: val,
		})
	}

	return

}

func processSerumOrdersExecuted(ordersExecuted []*orderExecutedEvent) (out []*kvdb.KV, err error) {
	for _, executed := range ordersExecuted {
		zlog.Debug("serum order executed",
			zap.Stringer("market", executed.market),
			zap.Uint64("order_seq_num", executed.orderSeqNum),
			zap.Uint64("slot_num", executed.slotNumber),
			zap.Uint32("trx_idx", executed.trxIdx),
			zap.Uint32("inst_idx", executed.instIdx),
		)

		out = append(out, &kvdb.KV{
			Key: keyer.EncodeOrderExecute(executed.market, executed.slotNumber, uint64(executed.trxIdx), uint64(executed.instIdx), executed.orderSeqNum),
		})
	}

	return
}
