package serumhist

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	kvdb "github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

func (i Injector) processSerumFills(serumFills []*fill) (out []*kvdb.KV, err error) {
	for _, serumFill := range serumFills {
		trader, err := i.cache.getTrader(i.ctx, serumFill.tradingAccount)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve trader for trading key %q: %w", serumFill.tradingAccount.String(), err)
		}

		if trader == nil {
			zlog.Warn("unable to find trader for trading account, skipping fill",
				zap.Stringer("trading_account", serumFill.tradingAccount),
				zap.Uint64("slot_number", serumFill.slotNumber),
				zap.Uint32("trx_id", serumFill.trxIdx),
				zap.Uint32("inst_id", serumFill.instIdx),
				zap.Stringer("market", serumFill.market),
			)
			return nil, nil
		}

		// we need to make sure we assign the trader before we proto encode, not all the keys contains the trader
		serumFill.fill.Trader = trader.String()
		cnt, err := proto.Marshal(serumFill.fill)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal to fill: %w", err)
		}

		zlog.Debug("serum new fill",
			zap.Stringer("side", serumFill.fill.Side),
			zap.Stringer("market", serumFill.market),
			zap.Stringer("trader", trader),
			zap.Stringer("trading_Account", serumFill.tradingAccount),
			zap.Uint64("order_seq_num", serumFill.orderSeqNum),
			zap.Uint64("slot_num", serumFill.slotNumber),
		)

		out = append(out, []*kvdb.KV{
			{
				Key:   keyer.EncodeFill(serumFill.market, serumFill.slotNumber, uint64(serumFill.trxIdx), uint64(serumFill.instIdx), serumFill.orderSeqNum),
				Value: cnt,
			},
			{
				Key: keyer.EncodeFillByTrader(*trader, serumFill.market, serumFill.slotNumber, uint64(serumFill.trxIdx), uint64(serumFill.instIdx), serumFill.orderSeqNum),
			},
			{
				Key: keyer.EncodeFillByTraderMarket(*trader, serumFill.market, serumFill.slotNumber, uint64(serumFill.trxIdx), uint64(serumFill.instIdx), serumFill.orderSeqNum),
			},
		}...)
	}
	return
}

func processSerumOrdersCancelled(ordersCancel []*orderCancelled) (out []*kvdb.KV, err error) {
	for _, cancel := range ordersCancel {
		zlog.Debug("serum order cancelled",
			zap.Stringer("market", cancel.market),
			zap.Uint64("order_seq_num", cancel.orderSeqNum),
			zap.Uint64("slot_num", cancel.slotNumber),
			zap.Uint32("trx_idx", cancel.trxIdx),
			zap.Uint32("inst_idx", cancel.instIdx),
		)

		val, err := proto.Marshal(cancel.instRef)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal to fill: %w", err)
		}

		out = append(out, &kvdb.KV{
			Key:   keyer.EncodeOrderCancel(cancel.market, cancel.slotNumber, uint64(cancel.trxIdx), uint64(cancel.instIdx), cancel.orderSeqNum),
			Value: val,
		})
	}

	return
}

func processSerumOrdersClosed(ordersClosed []*orderClosed) (out []*kvdb.KV, err error) {
	for _, close := range ordersClosed {
		zlog.Debug("serum order closed",
			zap.Stringer("market", close.market),
			zap.Uint64("order_seq_num", close.orderSeqNum),
			zap.Uint64("slot_num", close.slotNumber),
			zap.Uint32("trx_idx", close.trxIdx),
			zap.Uint32("inst_idx", close.instIdx),
		)

		val, err := proto.Marshal(close.instRef)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal to fill: %w", err)
		}

		out = append(out, &kvdb.KV{
			Key:   keyer.EncodeOrderClose(close.market, close.slotNumber, uint64(close.trxIdx), uint64(close.instIdx), close.orderSeqNum),
			Value: val,
		})
	}

	return

}

func processSerumOrdersExecuted(ordersExecuted []*orderExecuted) (out []*kvdb.KV, err error) {
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
