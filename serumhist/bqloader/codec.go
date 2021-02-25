package bqloader

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/linkedin/goavro/v2"
)

var (
	codecOrders    *goavro.Codec
	codecOrderFill *goavro.Codec
	codecTraders   *goavro.Codec
)

type Encoder interface {
	Codec() *goavro.Codec
	Encode() map[string]interface{}
}

func init() {
	//var err error
	//ordersSchemaSpecification, err := schemas.GetAvroSchemaDefinition(tableOrders.String(), "v1")
	//if err != nil {
	//	panic(fmt.Sprintf("unable to parse AVRO schema for codecOrders: %s", err.Error()))
	//}
	//codecOrders, err = goavro.NewCodec(ordersSchemaSpecification)
	//if err != nil {
	//	panic(fmt.Sprintf("unable to parse AVRO schema for codecOrders: %s", err.Error()))
	//}
	//
	//fillsSchemaSpecification, err := schemas.GetAvroSchemaDefinition(tableFills.String(), "v1")
	//if err != nil {
	//	panic(fmt.Sprintf("unable to parse AVRO schema for codecOrderFill: %s", err.Error()))
	//}
	//codecOrderFill, err = goavro.NewCodec(fillsSchemaSpecification)
	//if err != nil {
	//	panic(fmt.Sprintf("unable to parse AVRO schema for codecOrderFill: %s", err.Error()))
	//}
	//
	//tradersSchemaSpecification, err := schemas.GetAvroSchemaDefinition(tableTraders.String(), "v1")
	//if err != nil {
	//	panic(fmt.Sprintf("unable to parse AVRO schema for codecTraders: %s", err.Error()))
	//}
	//codecTraders, err = goavro.NewCodec(tradersSchemaSpecification)
	//if err != nil {
	//	panic(fmt.Sprintf("unable to parse AVRO schema for codecTraders: %s", err.Error()))
	//}
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

func (e *newOrderEncoder) Codec() *goavro.Codec {
	return codecOrders
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

func (e *orderFillEncoder) Codec() *goavro.Codec {
	return codecOrderFill
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

func (e *tradingAccountEncoder) Codec() *goavro.Codec {
	return codecTraders
}

func (e *tradingAccountEncoder) Encode() map[string]interface{} {
	m := map[string]interface{}{
		"account":  e.Account.String(),
		"trader":   e.Trader.String(),
		"slot_num": int64(e.SlotNumber),
	}
	return m
}
