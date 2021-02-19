package bqloader

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/dfuse-io/solana-go"
	"go.uber.org/zap"
)

// 3 avro file handlers one per bucket
// each avro file will have his "start block"

type Mapper interface {
	Map(interface{}) map[string]interface{}
	AvroCodec() *goavro.Codec
	LogValues(interface{}) zap.Fields
	TableName() string
}

type orderMapper struct{
	codec *goavro.Codec
}

func (m orderMapper) AvroCodec() *goavro.Codec { return m.codec }

func (orderMapper) Map(obj interface{}) map[string]interface{} {
	el := obj.(*serumhist.Order)
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

func (orderMapper) LogValues(obj interface{}) zap.Fields {
	o := obj.(*serumhist.NewOrder)
}
func (order Mapper) TableName() string { return "orders" }

var supportedInjectionObjects = map[reflect.Type]struct{
	func ToAvro(interface{}) []byte

}

func init() {
	supportedInjectionObjects[reflect.Type(&serumhist.NewOrder{})] = orderMapper{goavro.newCodec(`{}`)}
	supportedInjectionObjects[reflect.Type(&serumhist.NewFille{})] = fillMapper{}
}

func (bq *BQLoader) dispatch(obj interface{}) error {
	handler := supportedInjectionObjects[reflect.Type(obj)]

	zlog.Debug("adding object", zap.String("type", reflect.Type(obj).String()), handler.LogFields()...)
	if err := bq.avroHandlers[tradingAccount].HandleEvent(TradingAccountToAvro(account, trader), slotNum, slotId); err != nil {
		return fmt.Errorf("unable to process trading account %w", err)
	}
	return nil

}

func (bq *BQLoader) processTradingAccount(account, trader solana.PublicKey, slotNum uint64, slotId string) error {
	zlog.Debug("serum trading account",
		zap.Stringer("account", account),
		zap.Stringer("trader", trader),
	)
	if err := bq.avroHandlers[tradingAccount].HandleEvent(TradingAccountToAvro(account, trader), slotNum, slotId); err != nil {
		return fmt.Errorf("unable to process trading account %w", err)
	}
	return nil
}

func (bq *BQLoader) processSerumNewOrders(events []*serumhist.NewOrder) error {
	handler := bq.avroHandlers[newOrder]
	for _, event := range events {
		zlog.Debug("serum new order",
			zap.Stringer("market", event.Market),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
		)

		if err := handler.HandleEvent(NewOrderToAvro(event), event.SlotNumber, event.SlotHash); err != nil {
			return fmt.Errorf("unable to process fill %w", err)
		}
	}
	return nil
}

func (bq *BQLoader) processSerumFills(events []*serumhist.FillEvent) error {
	handler := bq.avroHandlers[fillOrder]
	for _, event := range events {
		zlog.Debug("serum new fill",
			zap.Stringer("side", event.Fill.Side),
			zap.Stringer("market", event.Market),
			zap.Stringer("trading_Account", event.TradingAccount),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
		)
		if err := handler.HandleEvent(FillEventToAvro(event), event.SlotNumber, event.SlotHash); err != nil {
			return fmt.Errorf("unable to process fill %w", err)
		}
	}
	return nil
}
