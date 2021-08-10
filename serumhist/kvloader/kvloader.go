package kvloader

import (
	"context"
	"time"

	"github.com/dfuse-io/kvdb/store"
	"github.com/streamingfast/shutter"
)

type KVLoader struct {
	*shutter.Shutter
	ctx               context.Context
	kvdb              store.KVStore
	cache             *tradingAccountCache
	flushSlotInterval uint64
}

const (
	DatabaseTimeout = 10 * time.Minute
)

func NewLoader(ctx context.Context, kvdb store.KVStore, flushSlotInterval uint64) *KVLoader {
	return &KVLoader{
		Shutter:           shutter.New(),
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
