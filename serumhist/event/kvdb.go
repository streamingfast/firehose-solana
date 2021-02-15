package event

import (
	"context"
	"fmt"
	"time"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
)

type Kvdb struct {
	kvdb store.KVStore
}

const (
	DatabaseTimeout = 10 * time.Minute
)

func NewKvdb(kvdb store.KVStore) *Kvdb {
	return &Kvdb{
		kvdb: kvdb,
	}
}

func (b *Kvdb) NewOrder(ctx context.Context, event *NewOrder) error {
	panic("implement me")
}

func (b *Kvdb) Fill(ctx context.Context, e *Fill) error {
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

	for _, kv := range kvs {
		if err := b.kvdb.Put(ctx, kv.Key, kv.Value); err != nil {
			return fmt.Errorf("unable to write serumhist injector in kvdb: %w", err)
		}
	}
	return nil
}

func (b *Kvdb) OrderExecuted(ctx context.Context, event *OrderExecuted) error {
	kv := store.KV{
		Key: keyer.EncodeOrderExecute(event.Market, event.SlotNumber, uint64(event.TrxIdx), uint64(event.InstIdx), event.OrderSeqNum),
	}

	if err := b.kvdb.Put(ctx, kv.Key, kv.Value); err != nil {
		return fmt.Errorf("unable to write serumhist injector in kvdb: %w", err)
	}

	return nil
}

func (b *Kvdb) OrderClosed(ctx context.Context, event *OrderClosed) error {
	val, err := proto.Marshal(event.InstrRef)
	if err != nil {
		return fmt.Errorf("unable to marshal to fill: %w", err)
	}

	key := keyer.EncodeOrderClose(event.Market, event.SlotNumber, uint64(event.TrxIdx), uint64(event.InstIdx), event.OrderSeqNum)
	value := val

	if err := b.kvdb.Put(ctx, key, value); err != nil {
		return fmt.Errorf("unable to write serumhist injector in kvdb: %w", err)
	}
	return nil
}

func (b *Kvdb) OrderCancelled(ctx context.Context, event *OrderCancelled) error {
	val, err := proto.Marshal(event.InstrRef)
	if err != nil {
		return fmt.Errorf("unable to marshal to fill: %w", err)
	}

	key := keyer.EncodeOrderCancel(event.Market, event.SlotNumber, uint64(event.TrxIdx), uint64(event.InstIdx), event.OrderSeqNum)
	value := val

	if err := b.kvdb.Put(ctx, key, value); err != nil {
		return fmt.Errorf("unable to write serumhist injector in kvdb: %w", err)
	}

	return nil
}

func (b *Kvdb) WriteCheckpoint(ctx context.Context, checkpoint *pbserumhist.Checkpoint) error {
	key := keyer.EncodeCheckpoint()

	value, err := proto.Marshal(checkpoint)
	if err != nil {
		return err
	}

	if err := b.kvdb.Put(ctx, key, value); err != nil {
		return fmt.Errorf("unable to store checkpoint in kvdb: %w", err)
	}
	return nil
}

func (b *Kvdb) Checkpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {
	key := keyer.EncodeCheckpoint()

	ctx, cancel := context.WithTimeout(ctx, DatabaseTimeout)
	defer cancel()

	val, err := b.kvdb.Get(ctx, key)
	if err == store.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error while reading checkpoint: %w", err)
	}

	// Decode val as `pbaccounthist.ShardCheckpoint`
	out := &pbserumhist.Checkpoint{}
	if err := proto.Unmarshal(val, out); err != nil {
		return nil, err
	}

	return out, nil

}

func (b *Kvdb) Flush(ctx context.Context) error {
	return b.kvdb.FlushPuts(ctx)
}
