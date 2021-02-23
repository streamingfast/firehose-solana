package bqloader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/dfuse-io/dfuse-solana/registry"
	"github.com/dfuse-io/dstore"
)

const (
	newOrder       = "orders"
	fillOrder      = "fills"
	tradingAccount = "traders"
	markets        = "markets"
	tokens         = "tokens"
)

type BQLoader struct {
	ctx context.Context

	client         *bigquery.Client
	dataset        *bigquery.Dataset
	store          dstore.Store
	registryServer *registry.Server

	traderAccountCache *tradingAccountCache
	eventHandlers      map[string]*eventHandler
}

func New(ctx context.Context, scratchSpaceDir string, storeUrl string, store dstore.Store, dataset *bigquery.Dataset, client *bigquery.Client, registry *registry.Server) *BQLoader {
	eventHandlers := make(map[string]*eventHandler)
	eventHandlers[newOrder] = newEventHandler(scratchSpaceDir, dataset, storeUrl, store, newOrder, CodecNewOrder)
	eventHandlers[fillOrder] = newEventHandler(scratchSpaceDir, dataset, storeUrl, store, fillOrder, CodecOrderFill)
	eventHandlers[tradingAccount] = newEventHandler(scratchSpaceDir, dataset, storeUrl, store, tradingAccount, CodecTraderAccount)

	cacheTableName := fmt.Sprintf("%s.serum.%s", dataset.ProjectID, tradingAccount)
	bq := &BQLoader{
		ctx:                ctx,
		client:             client,
		dataset:            dataset,
		store:              store,
		eventHandlers:      eventHandlers,
		registryServer:     registry,
		traderAccountCache: newTradingAccountCache(cacheTableName, client),
	}

	return bq
}

func (bq *BQLoader) PrimeTradeCache(ctx context.Context) error {
	zlog.Info("priming bq trader cache")
	return bq.traderAccountCache.load(ctx)
}

//shutdown all handlers.  collect any errors into a single error value
func (bq *BQLoader) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(len(bq.eventHandlers))

	errChan := make(chan error)
	go func() {
		wg.Wait()
		close(errChan)
	}()

	for _, h := range bq.eventHandlers {
		go func(handler *eventHandler) {
			defer wg.Done()
			err := handler.Shutdown(shutdownCtx)
			if err != nil {
				errChan <- err
			}
		}(h)
	}

	var err error
	for e := range errChan {
		if e == nil {
			continue
		}
		if err == nil {
			err = e
			continue
		}
		err = fmt.Errorf("%s, %s", err.Error(), e.Error())
	}

	return err
}
