package bqloader

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/bigquery"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

type tradingAccountCache struct {
	lock     sync.RWMutex
	table    string
	bqClient *bigquery.Client
	accounts map[string]string
}

func newTradingAccountCache(tableName string, client *bigquery.Client) *tradingAccountCache {
	return &tradingAccountCache{
		table:    tableName,
		bqClient: client,
		accounts: map[string]string{},
	}
}

func (t *tradingAccountCache) load(ctx context.Context) {
	t.lock.Lock()
	defer t.lock.Unlock()

	q := t.bqClient.Query(fmt.Sprintf("SELECT * FROM `%s`", t.table))
	j, err := q.Run(ctx)
	if err != nil {

	}
	it, err := j.Read(ctx)
	if err != nil {

	}

	type AccountTraderRow struct {
		Account string `bigquery:"account"`
		Trader  string `bigquery:"trader"`
	}

	for {
		var row AccountTraderRow
		err := it.Next(&row)
		if err == iterator.Done {
			return
		}
		if err != nil {
			zlog.Error("could not load trader cache", zap.Error(err))
			return
		}
		t.accounts[row.Account] = row.Trader
	}
}

func (t *tradingAccountCache) setTradingAccount(account, trader string) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, found := t.accounts[account]; found {
		return nil
	}

	t.accounts[account] = trader
	return nil
}

func (t *tradingAccountCache) getTrader(tradingAccount string) (trader string, found bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	trader, found = t.accounts[tradingAccount]
	return
}
