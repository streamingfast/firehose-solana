package kv

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/solanadb"
	"github.com/dfuse-io/kvdb/store"
	"go.uber.org/zap"
)

type DB struct {
	store  store.KVStore
	logger *zap.Logger
}

func init() {
	solanadbFactory := func(dsn string) (solanadb.DB, error) {
		return New(dsn)
	}

	solanadb.Register("badger", solanadbFactory)
	solanadb.Register("tikv", solanadbFactory)
	solanadb.Register("bigkv", solanadbFactory)
	solanadb.Register("cznickv", solanadbFactory)
}

func New(dsn string) (*DB, error) {
	zlog.Debug("creating kv db", zap.String("dsn", dsn))
	db := &DB{
		logger: zap.NewNop(),
	}

	kvStore, err := newCachedKVDB(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable retrieve kvdb driver: %w", err)
	}

	db.store = kvStore

	return db, nil
}

func (db *DB) Close() error {
	return db.store.Close()
}
