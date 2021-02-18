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
	newOrder       = "serum-orders"
	fillOrder      = "serum-fills"
	tradingAccount = "serum-traders"
)

type BQLoader struct {
	ctx     context.Context
	dataset *bigquery.Dataset
	store   dstore.Store

	avroHandlers map[string]*avroHandler
	tables       []string
}

func New(ctx context.Context, scratchSpaceDir string, client *bigquery.Client, store dstore.Store, datasetName string) *BQLoader {
	avroHandlers := make(map[string]*avroHandler)
	avroHandlers[newOrder] = NewAvroHandler(scratchSpaceDir, store, newOrder, CodecNewOrder)
	avroHandlers[fillOrder] = NewAvroHandler(scratchSpaceDir, store, fillOrder, CodecOrderFill)
	avroHandlers[tradingAccount] = NewAvroHandler(scratchSpaceDir, store, tradingAccount, CodecTraderAccount)

	tables := []string{newOrder, fillOrder, tradingAccount}

	bq := &BQLoader{
		ctx:          ctx,
		dataset:      client.Dataset(datasetName),
		store:        store,
		avroHandlers: avroHandlers,
		tables:       tables,
	}
	return bq
}

func (bq *BQLoader) StartLoaders(ctx context.Context) {
	for _, tableName := range bq.tables {
		ref := bigquery.NewGCSReference(bq.store.ObjectPath(tableName))

		loader := bq.dataset.Table(tableName).LoaderFrom(ref)
		loader.UseAvroLogicalTypes = true
		go func(l *bigquery.Loader) {
			_, _ = l.Run(ctx) // TODO: handle these?
		}(loader)
	}
}

//shutdown all avro handlers.  collect any errors into a single error value
func (bq *BQLoader) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(len(bq.avroHandlers))

	errChan := make(chan error)
	go func() {
		wg.Wait()
		close(errChan)
	}()

	for _, h := range bq.avroHandlers {
		go func(handler *avroHandler) {
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
