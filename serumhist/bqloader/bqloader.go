package bqloader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/dfuse-io/dstore"
)

const (
	newOrder       = "orders"
	fillOrder      = "fills"
	tradingAccount = "traders"
)

type BQLoader struct {
	ctx     context.Context
	dataset *bigquery.Dataset
	store   dstore.Store

	eventHandlers map[string]*eventHandler
}

func New(ctx context.Context, scratchSpaceDir string, storeUrl string, store dstore.Store, dataset *bigquery.Dataset) *BQLoader {
	eventHandlers := make(map[string]*eventHandler)
	eventHandlers[newOrder] = newEventHandler(scratchSpaceDir, dataset, storeUrl, store, newOrder, CodecNewOrder)
	eventHandlers[fillOrder] = newEventHandler(scratchSpaceDir, dataset, storeUrl, store, fillOrder, CodecOrderFill)
	eventHandlers[tradingAccount] = newEventHandler(scratchSpaceDir, dataset, storeUrl, store, tradingAccount, CodecTraderAccount)

	bq := &BQLoader{
		ctx:           ctx,
		dataset:       dataset,
		store:         store,
		eventHandlers: eventHandlers,
	}

	return bq
}

//shutdown all avro handlers.  collect any errors into a single error value
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
