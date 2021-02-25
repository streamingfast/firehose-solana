package bqloader

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist"
	"go.uber.org/zap"
)

type Encoder interface {
	Encode() map[string]interface{}
	Log()
}

func AsEncoder(i interface{}) Encoder {
	switch v := i.(type) {
	case *serumhist.NewOrder:
		return &newOrderEncoder{v}
	case *serumhist.FillEvent:
		return &orderFillEncoder{v}
	case *serumhist.TradingAccount:
		return &tradingAccountEncoder{v}
	default:
		panic(fmt.Sprintf("Encoder not supported for type %s", v))
	}
	return nil
}

type newOrderEncoder struct {
	*serumhist.NewOrder
}

func (e *newOrderEncoder) Log() {
	zlog.Debug("serum new order",
		zap.Stringer("market", e.Market),
		zap.Uint64("order_seq_num", e.OrderSeqNum),
		zap.Uint64("slot_num", e.SlotNumber),
	)
}

func (e *newOrderEncoder) Encode() map[string]interface{} {
	m := map[string]interface{}{
		"num":          int64(e.Order.Num),
		"market":       e.Ref.Market.String(),
		"trader":       e.Trader.String(),
		"side":         e.Order.Side.String(),
		"limit_price":  int64(e.Order.LimitPrice),
		"max_quantity": int64(e.Order.MaxQuantity),
		"type":         e.Order.Type.String(),
		"slot_num":     int64(e.Ref.SlotNumber),
		"slot_hash":    e.Order.SlotHash,
		"trx_id":       e.Order.TrxId,
		"trx_idx":      int32(e.Ref.TrxIdx),
		"inst_idx":     int32(e.Ref.InstIdx),
	}
	return m
}

type orderFillEncoder struct {
	*serumhist.FillEvent
}

func (e *orderFillEncoder) Log() {
	zlog.Debug("serum new fill",
		zap.Stringer("side", e.Fill.Side),
		zap.Stringer("market", e.Market),
		zap.Stringer("trading_Account", e.TradingAccount),
		zap.Uint64("order_seq_num", e.OrderSeqNum),
		zap.Uint64("slot_num", e.SlotNumber),
	)
}

func (e *orderFillEncoder) Encode() map[string]interface{} {
	m := map[string]interface{}{
		"trader":               e.TradingAccount.String(),
		"market":               e.Ref.Market.String(),
		"order_id":             e.Fill.OrderId,
		"side":                 e.Fill.Side.String(),
		"maker":                e.Fill.Maker,
		"native_qty_paid":      int64(e.Fill.NativeQtyPaid),
		"native_qty_received":  int64(e.Fill.NativeQtyReceived),
		"native_fee_or_rebate": int64(e.Fill.NativeFeeOrRebate),
		"fee_tier":             e.Fill.FeeTier.String(),
		"timestamp":            e.Fill.Timestamp.AsTime(),
		"slot_num":             int64(e.Ref.SlotNumber),
		"slot_hash":            e.Ref.SlotHash,
		"trx_id":               e.Fill.TrxId,
		"trx_idx":              int32(e.Ref.TrxIdx),
		"inst_idx":             int32(e.Ref.InstIdx),
		"order_seq_num":        int64(e.Ref.OrderSeqNum),
	}
	return m
}

type tradingAccountEncoder struct {
	*serumhist.TradingAccount
}

func (e *tradingAccountEncoder) Log() {
	zlog.Debug("serum trading account",
		zap.Stringer("account", e.Account),
		zap.Stringer("trader", e.Trader),
	)
}

func (e *tradingAccountEncoder) Encode() map[string]interface{} {
	m := map[string]interface{}{
		"account":  e.Account.String(),
		"trader":   e.Trader.String(),
		"slot_num": int64(e.SlotNumber),
	}
	return m
}
