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
			{"name": "timestamp", "type": {"type": "long", "logicalType" : "timestamp-millis"}},
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

func FillEventToAvro(e *serumhist.FillEvent) map[string]interface{} {
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

func TradingAccountToAvro(tradingAccount, trader solana.PublicKey) map[string]interface{} {
	m := map[string]interface{}{
		"account": tradingAccount.String(),
		"trader":  trader.String(),
	}
	return m
}
