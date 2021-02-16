package bqloader

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	"github.com/dfuse-io/dstore"
	"sync"
	"time"
)

const (
	newOrder       = "serum-orders"
	fillOrder      = "serum-fills"
	tradingAccount = "serum-traders"
)

type BQLoader struct {
	ctx     context.Context
	dataset string

	avroHandlers map[string]*avroHandler
}

func New(ctx context.Context, client *bigquery.Client, dataset string) *BQLoader {
	loader := &BQLoader{
		ctx:     ctx,
		dataset: dataset,
	}

	var store dstore.Store
	var fileDir, fileName string

	avroHandlers := make(map[string]*avroHandler)
	avroHandlers[newOrder] = NewAvroHandler(fileDir, fileName, store, newOrder, OrderCreatedCodec)
	avroHandlers[fillOrder] = NewAvroHandler(fileDir, fileName, store, fillOrder, OrderFilledCodec)
	avroHandlers[tradingAccount] = NewAvroHandler(fileDir, fileName, store, tradingAccount, TraderAccountCodec)

	return loader
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
		if err == nil {
			err = e
		}
		err = fmt.Errorf("%s, %s", err.Error(), e.Error())
	}

	return err
}
