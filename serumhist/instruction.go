package serumhist

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"

	"github.com/golang/protobuf/proto"

	kvdb "github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

func (i *Injector) processSerumFill(ctx context.Context, serumFill *serumFill) error {
	trader, err := i.cache.getTrader(ctx, serumFill.tradingAccount)
	if err != nil {
		return fmt.Errorf("unable to retrieve trader for trading key %q: %w", serumFill.tradingAccount.String(), err)
	}

	if trader == nil {
		zlog.Warn("unable to find trader for trading account, skipping fill",
			zap.Stringer("trading_account", serumFill.tradingAccount),
			zap.Uint64("slot_number", serumFill.slotNumber),
			zap.Uint64("trx_id", serumFill.trxIdx),
			zap.Uint64("inst_id", serumFill.instIdx),
			zap.Stringer("market", serumFill.market),
		)
		return nil
	}

	cnt, err := proto.Marshal(serumFill.fill)
	if err != nil {
		return fmt.Errorf("unable to marshal to fill: %w", err)
	}

	zlog.Debug("serum new fill",
		zap.Stringer("side", serumFill.fill.Side),
		zap.Stringer("market", serumFill.market),
		zap.Stringer("trader", trader),
		zap.Stringer("trading_Account", serumFill.tradingAccount),
		zap.Uint64("order_seq_num", serumFill.orderSeqNum),
		zap.Uint64("slot_num", serumFill.slotNumber),
	)

	kvs := []*kvdb.KV{
		{
			Key:   keyer.EncodeFillByTrader(*trader, serumFill.market, serumFill.slotNumber, serumFill.trxIdx, serumFill.instIdx, serumFill.orderSeqNum),
			Value: cnt,
		},
		{
			Key:   keyer.EncodeFillByMarketTrader(*trader, serumFill.market, serumFill.slotNumber, serumFill.trxIdx, serumFill.instIdx, serumFill.orderSeqNum),
			Value: cnt,
		},
	}

	for _, kv := range kvs {
		if err := i.kvdb.Put(ctx, kv.Key, kv.Value); err != nil {
			return fmt.Errorf("unable to write serumhist fill in kvdb: %w", err)
		}
	}
	return nil
}
