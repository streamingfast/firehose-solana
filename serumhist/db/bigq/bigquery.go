package bigq

import (
	"cloud.google.com/go/bigquery"
	serumhistdb "github.com/dfuse-io/dfuse-solana/serumhist/db"
)

type Bigq struct {
	client *bigquery.Client

	orderCreatedMapping *Mapping
	orderCreatedTable   *bigquery.Table
	orderFilledMapping  *Mapping
	orderFilledTable    *bigquery.Table
	dataset             string
}

func New(client *bigquery.Client, dataset string) serumhistdb.DB {
	return &Bigq{
		client:  client,
		dataset: dataset,
	}
}

func (b *Bigq) Close() {
	_ = b.client.Close()
}
