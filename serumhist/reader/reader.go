package reader

import (
	"github.com/streamingfast/kvdb/store"
)

type Reader struct {
	store store.KVStore
}

func New(store store.KVStore) *Reader {
	return &Reader{
		store: store,
	}
}
