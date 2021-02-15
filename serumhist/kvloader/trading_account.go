package kvloader

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"go.uber.org/zap"
)

type tradingAccountCache struct {
	kvdb     store.KVStore
	accounts map[string]solana.PublicKey
}

func newTradingAccountCache(kvdb store.KVStore) *tradingAccountCache {
	return &tradingAccountCache{
		kvdb:     kvdb,
		accounts: map[string]solana.PublicKey{},
	}
}

func (t *tradingAccountCache) load(ctx context.Context) {
	zlog.Debug("loading known trading account cache")
	it := t.kvdb.Scan(ctx, keyer.StartOfTradingAccount(), keyer.EndOfTradingAccount(), 0)
	for it.Next() {
		tradingAccount := solana.PublicKeyFromBytes(it.Item().Key)
		trader := solana.PublicKeyFromBytes(it.Item().Value)
		t.accounts[tradingAccount.String()] = trader
	}
	metrics.TradingAccountCount.SetUint64(uint64(len(t.accounts)))
	serumhist.zlog.Debug("trading account cache loaded",
		zap.Int("account_count", len(t.accounts)),
	)
}

func (t *tradingAccountCache) setTradingAccount(ctx context.Context, tradingAccount, trader solana.PublicKey) error {
	if _, found := t.accounts[tradingAccount.String()]; found {
		if serumhist.traceEnabled {
			serumhist.zlog.Debug("found trading account skipping the setting it to kvdb",
				zap.Stringer("trader", trader),
				zap.Stringer("trading_acount", tradingAccount),
			)
		}
		return nil
	}

	t.accounts[tradingAccount.String()] = trader
	key := keyer.EncodeTradingAccount(tradingAccount)
	if err := t.kvdb.Put(ctx, key, trader[:]); err != nil {
		return fmt.Errorf("error setting trading account: %w", err)
	}
	metrics.TradingAccountCount.SetUint64(uint64(len(t.accounts)))
	return nil
}

func (t *tradingAccountCache) getTrader(ctx context.Context, tradingAccount solana.PublicKey) (*solana.PublicKey, error) {
	if trader, found := t.accounts[tradingAccount.String()]; found {
		return &trader, nil
	}
	key := keyer.EncodeTradingAccount(tradingAccount)
	val, err := t.kvdb.Get(ctx, key)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("error retriving trading account: %w", err)
	}

	p := solana.PublicKeyFromBytes(val)
	return &p, nil
}
