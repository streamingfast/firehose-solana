package bqloader

import (
	"context"
	"github.com/dfuse-io/dstore"

	"cloud.google.com/go/bigquery"
)

const (
	newOrder       = "ORDER_CREATED"
	fillOrder      = "ORDER_FILLED"
	tradingAccount = "TRADING_ACCOUNT"
)

type BQLoader struct {
	ctx      context.Context
	bqclient *bigquery.Client // ???/
	dataset  string

	avroHandlers map[string]*avroHandler
}

func New(ctx context.Context, client *bigquery.Client, dataset string) *BQLoader {
	loader := &BQLoader{
		ctx:      ctx,
		bqclient: client,
		dataset:  dataset,
	}

	var store dstore.Store
	var fileDir, fileName string

	avroHandlers := make(map[string]*avroHandler)
	avroHandlers[newOrder] = NewAvroHandler(fileDir, fileName, store, OrderCreatedCodec)
	avroHandlers[fillOrder] = NewAvroHandler(fileDir, fileName, store, OrderFilledCodec)
	avroHandlers[tradingAccount] = NewAvroHandler(fileDir, fileName, store, TraderAccountCodec)

	return loader
}

func (bq *BQLoader) Close() error {

	return bq.bqclient.Close()
}
