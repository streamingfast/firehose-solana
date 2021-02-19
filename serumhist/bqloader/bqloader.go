package bqloader

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/dfuse-io/dstore"
	"go.uber.org/zap"
	"google.golang.org/api/googleapi"
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

	avroHandlers map[string]*avroHandler
	tables       []string
}

func New(ctx context.Context, scratchSpaceDir string, store dstore.Store, dataset *bigquery.Dataset) *BQLoader {
	tables := []string{newOrder, fillOrder, tradingAccount}

	avroHandlers := make(map[string]*avroHandler)
	avroHandlers[newOrder] = NewAvroHandler(scratchSpaceDir, store, newOrder, CodecNewOrder)
	avroHandlers[fillOrder] = NewAvroHandler(scratchSpaceDir, store, fillOrder, CodecOrderFill)
	avroHandlers[tradingAccount] = NewAvroHandler(scratchSpaceDir, store, tradingAccount, CodecTraderAccount)

	bq := &BQLoader{
		tables:       tables,
		ctx:          ctx,
		dataset:      dataset,
		store:        store,
		avroHandlers: avroHandlers,
	}

	return bq
}

func (bq *BQLoader) StartLoaders(ctx context.Context, storeUrl string) {
	for _, tableName := range bq.tables {
		err := bq.dataset.Table(tableName).Create(bq.ctx, nil)
		if err, ok := err.(*googleapi.Error); !ok || err.Code != 409 { // ignore already-exists error
			zlog.Error("could not create table", zap.Error(err))
		}

		go func(table string) {
			uri := strings.Join([]string{storeUrl, table, "*"}, "/")
			ref := bigquery.NewGCSReference(uri)
			ref.SourceFormat = bigquery.Avro
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Minute):
				}

				loader := bq.dataset.Table(table).LoaderFrom(ref)
				loader.UseAvroLogicalTypes = true
				job, err := loader.Run(ctx)
				if err != nil {
					zlog.Error("could not run loader", zap.String("table", table), zap.Error(err))
					continue
				}
				js, err := job.Wait(ctx)
				if err != nil {
					zlog.Error("could not create loader job", zap.Error(err))
					continue
				}
				if js.Err() != nil {
					zlog.Error("could not run loader job", zap.Error(js.Err()))
					continue
				}
			}
		}(tableName)
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
