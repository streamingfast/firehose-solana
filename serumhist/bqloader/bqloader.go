package bqloader

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dstore"
	"net/url"
	"sync"
	"time"
)

const (
	newOrder       = "serum-orders"
	fillOrder      = "serum-fills"
	tradingAccount = "serum-traders"
)

type BQLoader struct {
	dataset *bigquery.Dataset
	store   dstore.Store

	avroHandlers  map[string]*avroHandler
	tables        []string
	loaderCleanup func()
}

func New(ctx context.Context, client *bigquery.Client, storeURL string, datasetName string) *BQLoader {
	var store dstore.Store

	gsUrl, err := url.Parse(storeURL)
	derr.Check("big_query_store_url", err)

	store, err = dstore.NewGSStore(gsUrl, "", "", true)
	derr.Check("big_query_store", err)

	var fileDir, fileName string
	avroHandlers := make(map[string]*avroHandler)
	avroHandlers[newOrder] = NewAvroHandler(fileDir, fileName, store, newOrder, OrderCreatedCodec)
	avroHandlers[fillOrder] = NewAvroHandler(fileDir, fileName, store, fillOrder, OrderFilledCodec)
	avroHandlers[tradingAccount] = NewAvroHandler(fileDir, fileName, store, tradingAccount, TraderAccountCodec)

	tables := []string{newOrder, fillOrder, tradingAccount}

	bq := &BQLoader{
		dataset:      client.Dataset(datasetName),
		store:        store,
		avroHandlers: avroHandlers,
		tables:       tables,
	}
	bq.Start(ctx)

	return bq
}

func (bq *BQLoader) Start(ctx context.Context) {
	for _, tableName := range bq.tables {
		ref := bigquery.NewGCSReference(bq.store.ObjectPath(tableName))

		loader := bq.dataset.Table(tableName).LoaderFrom(ref)
		go func(l *bigquery.Loader) {
			_, _ = l.Run(ctx) // TODO: handle these?
		}(loader)
	}
}

//shutdown all avro handlers.  collect any errors into a single error value
func (bq *BQLoader) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errChan := make(chan error)

	wg := sync.WaitGroup{}
	wg.Add(len(bq.avroHandlers))

	for _, h := range bq.avroHandlers {
		go func(handler *avroHandler) {
			defer wg.Done()
			err := handler.Shutdown(shutdownCtx)
			if err != nil {
				errChan <- err
			}
		}(h)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

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

	// stop loaders
	bq.loaderCleanup()

	return err
}
