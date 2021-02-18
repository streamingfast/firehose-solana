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