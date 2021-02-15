package kvdb

import (
	"time"

	serumhistdb "github.com/dfuse-io/dfuse-solana/serumhist/db"
	"github.com/dfuse-io/kvdb/store"
)

type Kvdb struct {
	kvdb store.KVStore
}

const (
	DatabaseTimeout = 10 * time.Minute
)

func New(kvdb store.KVStore) serumhistdb.DB {
	return &Kvdb{
		kvdb: kvdb,
	}
}

func (kv *Kvdb) Close() {
	kv.kvdb.Close()
}
