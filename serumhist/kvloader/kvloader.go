package kvloader

import (
	"context"
	"time"

	"github.com/dfuse-io/kvdb/store"
)

type KVLoader struct {
	kvdb              store.KVStore
	ctx               context.Context
	cache             *tradingAccountCache
	flushSlotInterval uint64
}

const (
	DatabaseTimeout = 10 * time.Minute
)

func NewLoader(ctx context.Context, kvdb store.KVStore, flushSlotInterval uint64) *KVLoader {
	return &KVLoader{
		ctx:               ctx,
		flushSlotInterval: flushSlotInterval,
		kvdb:              kvdb,
		cache:             newTradingAccountCache(kvdb),
	}
}

func (kv *KVLoader) PrimeTradeCache(ctx context.Context) {
	zlog.Info("priming kvdb cache")
	kv.cache.load(ctx)
}

func (kv *KVLoader) Close() error {
	return kv.kvdb.Close()
}
