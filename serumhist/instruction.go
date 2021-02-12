package serumhist

import (
	"context"
	"fmt"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	kvdb "github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
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

	// we need to make sure we assign the trader before we proto encode, not all the keys contains the trader
	serumFill.fill.Trader = trader.String()
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
			Key:   keyer.EncodeFill(serumFill.market, serumFill.slotNumber, serumFill.trxIdx, serumFill.instIdx, serumFill.orderSeqNum),
			Value: cnt,
		},
		{
			Key: keyer.EncodeFillByTrader(*trader, serumFill.market, serumFill.slotNumber, serumFill.trxIdx, serumFill.instIdx, serumFill.orderSeqNum),
		},
		{
			Key: keyer.EncodeFillByTraderMarket(*trader, serumFill.market, serumFill.slotNumber, serumFill.trxIdx, serumFill.instIdx, serumFill.orderSeqNum),
		},
	}

	for _, kv := range kvs {
		if err := i.kvdb.Put(ctx, kv.Key, kv.Value); err != nil {
			return fmt.Errorf("unable to write serumhist fill in kvdb: %w", err)
		}
	}
	return nil
}

func (i *Injector) processSerumCancel(ctx context.Context, serumCancel *serumOrderCancelled) error {
	zlog.Debug("serum new cancel",
		zap.Stringer("market", serumCancel.market),
		zap.Uint64("order_seq_num", serumCancel.orderSeqNum),
		zap.Uint64("slot_num", serumCancel.slotNumber),
		zap.Uint64("trx_idx", serumCancel.trxIdx),
		zap.Uint64("inst_idx", serumCancel.instIdx),
	)

	tmporal := &pbserumhist.SerumTemporal{
		SlotNum:              serumCancel.slotNumber,
		TrxHash:              "",
		TrxIdx:               serumCancel.trxIdx,
		InstIdx:              serumCancel.instIdx,
		SlotHash:             "",
		Timestamp:            timestamppb.Now(),
	}

	val, err := proto.Marshal(tmporal)
	if err != nil {
		return fmt.Errorf("unable to marshal to fill: %w", err)
	}

	kvs := []*kvdb.KV{
		{
			Key:   keyer.EncodeOrderCancel(serumCancel.market, serumCancel.slotNumber, serumCancel.trxIdx, serumCancel.instIdx, serumCancel.orderSeqNum),
			Value: val,
		},
	}

	for _, kv := range kvs {
		if err := i.kvdb.Put(ctx, kv.Key, kv.Value); err != nil {
			return fmt.Errorf("unable to write serumhist fill in kvdb: %w", err)
		}
	}
	return nil
}

func (i *Injector) processSerumExecute(ctx context.Context, serumExecute *serumOrderExecuted) error {
	zlog.Debug("serum new execute",
		zap.Stringer("market", serumExecute.market),
		zap.Uint64("order_seq_num", serumExecute.orderSeqNum),
		zap.Uint64("slot_num", serumExecute.slotNumber),
		zap.Uint64("trx_idx", serumExecute.trxIdx),
		zap.Uint64("inst_idx", serumExecute.instIdx),
	)

	tmporal := &pbserumhist.SerumTemporal{
		SlotNum:   serumExecute.slotNumber,
		TrxHash:   "",
		TrxIdx:    uint32(serumExecute.trxIdx),
		InstIdx:   uint32(serumExecute.instIdx),
		SlotHash:  "",
		Timestamp: timestamppb.Now(),
	}

	val, err := proto.Marshal(tmporal)
	if err != nil {
		return fmt.Errorf("unable to marshal to fill: %w", err)
	}

	kvs := []*kvdb.KV{
		{
			Key:   keyer.EncodeOrder(serumExecute.market, serumExecute.slotNumber, serumExecute.trxIdx, serumExecute.instIdx, serumExecute.orderSeqNum),
		},
	}

	for _, kv := range kvs {
		if err := i.kvdb.Put(ctx, kv.Key, kv.Value); err != nil {
			return fmt.Errorf("unable to write serumhist fill in kvdb: %w", err)
		}
	}
	return nil
}