package kvloader

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/dfuse-solana/serumhist"

	"go.uber.org/zap"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"
	"github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
)

func (kv *KVLoader) writeNewOrder(event *serumhist.NewOrder) error {
	panic("implement me")
}

func (kv *KVLoader) writeFill(e *serumhist.FillEvent) error {
	ctx := kv.ctx
	cnt, err := proto.Marshal(e.Fill)
	if err != nil {
		return fmt.Errorf("unable to marshal to fill: %w", err)
	}
	kvs := []*store.KV{
		{
			Key:   keyer.EncodeFill(e.Market, e.SlotNumber, uint64(e.TrxIdx), uint64(e.InstIdx), e.OrderSeqNum),
			Value: cnt,
		},
		{
			Key: keyer.EncodeFillByTrader(e.Trader, e.Market, e.SlotNumber, uint64(e.TrxIdx), uint64(e.InstIdx), e.OrderSeqNum),
		},
		{
			Key: keyer.EncodeFillByTraderMarket(e.Trader, e.Market, e.SlotNumber, uint64(e.TrxIdx), uint64(e.InstIdx), e.OrderSeqNum),
		},
	}

	for _, k := range kvs {
		if err := kv.kvdb.Put(ctx, k.Key, k.Value); err != nil {
			return fmt.Errorf("unable to write serumhist injector in kvdb: %w", err)
		}
	}
	return nil
}

func (kv *KVLoader) writeOrderExecuted(event *serumhist.OrderExecuted) error {
	ctx := kv.ctx
	k := store.KV{
		Key: keyer.EncodeOrderExecute(event.Market, event.SlotNumber, uint64(event.TrxIdx), uint64(event.InstIdx), event.OrderSeqNum),
	}

	if err := kv.kvdb.Put(ctx, k.Key, k.Value); err != nil {
		return fmt.Errorf("unable to write serumhist injector in kvdb: %w", err)
	}

	return nil
}

func (kv *KVLoader) writeOrderClosed(event *serumhist.OrderClosed) error {
	ctx := kv.ctx
	val, err := proto.Marshal(event.InstrRef)
	if err != nil {
		return fmt.Errorf("unable to marshal to fill: %w", err)
	}

	key := keyer.EncodeOrderClose(event.Market, event.SlotNumber, uint64(event.TrxIdx), uint64(event.InstIdx), event.OrderSeqNum)
	value := val

	if err := kv.kvdb.Put(ctx, key, value); err != nil {
		return fmt.Errorf("unable to write serumhist injector in kvdb: %w", err)
	}
	return nil
}

func (kv *KVLoader) writeOrderCancelled(event *serumhist.OrderCancelled) error {
	ctx := kv.ctx
	val, err := proto.Marshal(event.InstrRef)
	if err != nil {
		return fmt.Errorf("unable to marshal to fill: %w", err)
	}

	key := keyer.EncodeOrderCancel(event.Market, event.SlotNumber, uint64(event.TrxIdx), uint64(event.InstIdx), event.OrderSeqNum)
	value := val

	if err := kv.kvdb.Put(ctx, key, value); err != nil {
		return fmt.Errorf("unable to write serumhist injector in kvdb: %w", err)
	}

	return nil
}

func (kv *KVLoader) writeCheckpoint(checkpoint *pbserumhist.Checkpoint) error {
	key := keyer.EncodeCheckpoint()

	value, err := proto.Marshal(checkpoint)
	if err != nil {
		return err
	}

	if err := kv.kvdb.Put(kv.ctx, key, value); err != nil {
		return fmt.Errorf("unable to store checkpoint in kvdb: %w", err)
	}
	return nil
}

func (kv *KVLoader) flush(ctx context.Context) error {
	return kv.kvdb.FlushPuts(ctx)
}

func (kv *KVLoader) flushIfNeeded(slotNum uint64, slotTime time.Time) error {
	batchSizeReached := slotNum%kv.flushSlotInterval == 0
	closeToHeadBlockTime := time.Since(slotTime) < 25*time.Second

	if batchSizeReached || closeToHeadBlockTime {
		reason := "needed"
		if batchSizeReached {
			reason += ", batch size reached"
		}

		if closeToHeadBlockTime {
			reason += ", close to head block"
		}

		err := kv.doFlush(slotNum, reason)
		if err != nil {
			return err
		}
		metrics.HeadBlockNumber.SetUint64(slotNum)
	}

	return nil
}

func (kv *KVLoader) doFlush(slotNum uint64, reason string) error {
	zlog.Debug("flushing block",
		zap.Uint64("slot_num", slotNum),
		zap.String("reason", reason),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	err := kv.flush(ctx)
	if err != nil {
		return fmt.Errorf("db flush: %w", err)
	}
	return nil
}
