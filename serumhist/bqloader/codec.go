package bqloader

import (
	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/dfuse-io/solana-go"
	"github.com/linkedin/goavro/v2"
)

var (
	CodecNewOrder      *goavro.Codec
	CodecOrderFill     *goavro.Codec
	CodecTraderAccount *goavro.Codec
)

func init() {
	var err error
	CodecNewOrder, err = goavro.NewCodec(`{
		"namespace": "io.dfuse",
		"type": "record",
		"name": "OrderFill",
		"fields": [
			{"name": "num", "type": "long"},
			{"name": "market", "type": "string"},
			{"name": "trader", "type": "string"},
			{"name": "side", "type": "string"},
			{"name": "limit_price", "type": "long"},
			{"name": "max_quantity", "type": "long"},
			{"name": "type", "type": "string"},
			{"name": "slot_num", "type": "long"},
			{"name": "slot_hash", "type": "string"},
			{"name": "trx_id", "type": "string"},
			{"name": "trx_idx", "type": "int"},
			{"name": "inst_idx", "type": "int"}
		]
	}`)
	if err != nil {
		panic("unable to parse AVRO schema for CodecNewOrder")
	}
	CodecOrderFill, err = goavro.NewCodec(`{
		"namespace": "io.dfuse",
		"type": "record",
		"name": "OrderFill",
		"fields": [
			{"name": "trader", "type": "string"},
			{"name": "market", "type": "string"},
			{"name": "order_id", "type": "string"},
			{"name": "side", "type": "string"},
			{"name": "maker", "type": "boolean"},
			{"name": "native_qty_paid", "type": "long"},
			{"name": "native_qty_received", "type": "long"},
			{"name": "native_fee_or_rebate", "type": "long"},
			{"name": "fee_tier", "type": "string"},
			{"name": "timestamp", "type": "long.timestamp-micros"},
			{"name": "slot_num", "type": "long"},
			{"name": "slot_hash", "type": "string"},
			{"name": "trx_id", "type": "string"},
			{"name": "trx_idx", "type": "int"},
			{"name": "inst_idx", "type": "int"},
			{"name": "order_seq_num", "type": "long"}
		]
	}`)
	if err != nil {
		panic("unable to parse AVRO schema for CodecOrderFilled")
	}
	CodecTraderAccount, err = goavro.NewCodec(`{
		"namespace": "io.dfuse",
		"type": "record",
		"name": "TraderAccount",
		"fields": [{"name": "account", "type": "string"},{"name": "trader", "type": "string"}]
	}`)
	if err != nil {
		panic("unable to parse AVRO schema for CodecTraderAccount")
	}
}

func NewOrderToAvro(e *serumhist.NewOrder) map[string]interface{} {
	return map[string]interface{}{
		"num":          e.Order.Num,
		"market":       e.Order.Market,
		"trader":       e.Order.Trader,
		"side":         e.Order.Side.String(),
		"limit_price":  e.Order.LimitPrice,
		"max_quantity": e.Order.MaxQuantity,
		"type":         e.Order.Type.String(),
		"fills":        e.Order.Fills,
		"slot_num":     e.Order.SlotNum,
		"slot_hash":    e.Order.SlotHash,
		"trx_id":       e.Order.TrxId,
		"trx_idx":      e.Order.TrxIdx,
		"inst_idx":     e.Order.InstIdx,
	}
}

func FillEventToAvro(e *serumhist.FillEvent) map[string]interface{} {
	return map[string]interface{}{
		"trader":               e.Fill.Trader,
		"market":               e.Fill.Market,
		"order_id":             e.Fill.OrderId,
		"side":                 e.Fill.Side.String(),
		"maker":                e.Fill.Maker,
		"native_qty_paid":      e.Fill.NativeQtyPaid,
		"native_qty_received":  e.Fill.NativeQtyReceived,
		"native_fee_or_rebate": e.Fill.NativeFeeOrRebate,
		"fee_tier":             e.Fill.FeeTier.String(),
		"timestamp":            e.Fill.Timestamp,
		"slot_num":             e.Fill.SlotNum,
		"slot_hash":            e.Fill.SlotHash,
		"trx_id":               e.Fill.TrxId,
		"trx_idx":              e.Fill.TrxIdx,
		"inst_idx":             e.Fill.InstIdx,
		"order_seq_num":        e.Fill.OrderSeqNum,
	}
}

func TradingAccountToAvro(tradingAccount, trader solana.PublicKey) map[string]interface{} {
	return map[string]interface{}{
		"account": tradingAccount.String(),
		"trader":  trader.String(),
	}
}
