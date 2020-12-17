package serumhist

import (
	"context"
	"testing"

	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/solana-go"

	"github.com/stretchr/testify/assert"
)

func TestRWCache(t *testing.T) {
	kvStore, cleanup := getKVTestFactory(t)
	defer cleanup()
	rwCache := NewRWCache(kvStore)
	ctx := context.Background()

	pubkey := solana.MustPublicKeyFromBase58("13iGJcA4w5hcJZDjJbJQor1zUiDLE4jv2rMW9HkD5Eo1")
	market := solana.MustPublicKeyFromBase58("77jtrBDbUvwsdNKeq1ERUBcg8kk2hNTzf5E4iRihNgTh")

	rwCache.Put(ctx, keyer.EncodeOrdersByPubkey(pubkey, market, 1, 3), []byte{})
	rwCache.Put(ctx, keyer.EncodeOrdersByPubkey(pubkey, market, 2, 4), []byte{})
	rwCache.Put(ctx, keyer.EncodeOrdersByPubkey(pubkey, market, 3, 5), []byte{})

	expectedKeys := [][]byte{
		keyer.EncodeOrdersByPubkey(pubkey, market, 1, 3),
		keyer.EncodeOrdersByPubkey(pubkey, market, 2, 4),
		keyer.EncodeOrdersByPubkey(pubkey, market, 3, 5),
	}
	i := 0
	rwCache.OrderedPuts(func(sKey string, value []byte) error {
		assert.Equal(t, string(expectedKeys[i]), sKey)
		i += 1
		return nil
	})
}
