package bqloader

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/dfuse-io/solana-go"
	"go.uber.org/zap"
)

// 3 avro file handlers one per bucket
// each avro file will have his "start block"

func (bq *BQLoader) processTradingAccount(account, trader solana.PublicKey, slotNum uint64, slotId string) error {
	if err := bq.avroHandlers[tradingAccount].handleEvent(TraderAccountToAvro(account, trader), slotNum, slotId); err != nil {
		return fmt.Errorf("unable to process trading account %w", err)
	}

	return nil
}

func (bq *BQLoader) processSerumNewOrders(events []*serumhist.NewOrder) error {
	for _, event := range events {
		zlog.Debug("serum new order",
			zap.Stringer("market", event.Market),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
		)

		if err := bq.avroHandlers[newOrder].handleEvent(OrderCreatedEventToAvro(event), event.SlotNumber, event.SlotHash); err != nil {
			return fmt.Errorf("unable to process fill %w", err)
		}
	}
	return nil
}

func (bq *BQLoader) processSerumFills(events []*serumhist.FillEvent) error {
	for _, event := range events {
		zlog.Debug("serum new fill",
			zap.Stringer("side", event.Fill.Side),
			zap.Stringer("market", event.Market),
			zap.Stringer("trading_Account", event.TradingAccount),
			zap.Uint64("order_seq_num", event.OrderSeqNum),
			zap.Uint64("slot_num", event.SlotNumber),
		)

		if err := bq.avroHandlers[fillOrder].handleEvent(OrderFilledEventToAvro(event), event.SlotNumber, event.SlotHash); err != nil {
			return fmt.Errorf("unable to process fill %w", err)
		}
	}
	return nil
}

func OrderCreatedEventToAvro(e *serumhist.NewOrder) map[string]interface{} {
	panic("implement me")
}

func OrderFilledEventToAvro(e *serumhist.FillEvent) map[string]interface{} {
	panic("implement me")
}

func TraderAccountToAvro(tradingAccount, trader solana.PublicKey) map[string]interface{} {
	panic("implement me")
}
