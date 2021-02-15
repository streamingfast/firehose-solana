package kvloader

import "context"

type KVLoader struct {
	ctx   context.Context
	cache *tradingAccountCache
}
