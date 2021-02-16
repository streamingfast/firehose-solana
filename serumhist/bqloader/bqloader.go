package bqloader

import (
	"context"

	"cloud.google.com/go/bigquery"
)

type BQLoader struct {
	ctx      context.Context
	bqclient *bigquery.Client
	dataset  string

	orderCreatedMapping *Mapping
	orderCreatedTable   *bigquery.Table
	orderFilledMapping  *Mapping
	orderFilledTable    *bigquery.Table
}

func New(ctx context.Context, client *bigquery.Client, dataset string) *BQLoader {
	return &BQLoader{
		ctx:      ctx,
		bqclient: client,
		dataset:  dataset,
	}
}

func (bq *BQLoader) Close() error {
	return bq.bqclient.Close()
}
