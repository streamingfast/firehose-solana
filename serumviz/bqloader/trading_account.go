package bqloader

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

type tradingAccountCache struct {
	lock sync.RWMutex

	table    string
	accounts map[string]string

	bqClient *bigquery.Client
}

func newTradingAccountCache(tableName string, client *bigquery.Client) *tradingAccountCache {
	return &tradingAccountCache{
		table:    tableName,
		bqClient: client,
		accounts: map[string]string{},
	}
}

func (t *tradingAccountCache) load(ctx context.Context) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	queryString := fmt.Sprintf("SELECT account,trader FROM `%s`", t.table)
	q := t.bqClient.Query(queryString)
	j, err := q.Run(ctx)
	if err != nil {
		return fmt.Errorf("could not run query `%s`: %w", queryString, err)
	}
	it, err := j.Read(ctx)
	if err != nil {
		return fmt.Errorf("could not read query results: %w", err)
	}

	type AccountTraderRow struct {
		Account string `bigquery:"account"`
		Trader  string `bigquery:"trader"`
	}

	for {
		var row AccountTraderRow
		err := it.Next(&row)
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return fmt.Errorf("could not read account trader row: %w", err)
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

func (t *tradingAccountCache) getTrader(account string) (trader string, found bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	trader, found = t.accounts[account]
	return
}
