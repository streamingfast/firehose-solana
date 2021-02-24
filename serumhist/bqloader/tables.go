package bqloader

import (
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/dfuse-io/dfuse-solana/serumhist/bqloader/schemas"
	"google.golang.org/api/googleapi"
)

const (
	tableOrders  = "orders"
	tableFills   = "fills"
	tableTraders = "traders"
	tableMarkets = "markets"
	tableTokens  = "tokens"

	tableProcessedFiles = "processed_files"
)

func (bq *BQLoader) InitTables() (err error) {
	err = bq.initNewOrdersTable()
	if err != nil {
		return
	}

	err = bq.initOrderFillsTable()
	if err != nil {
		return
	}

	err = bq.initTradersTable()
	if err != nil {
		return
	}

	err = bq.initMarketsTable()
	if err != nil {
		return
	}

	err = bq.initTokensTable()
	if err != nil {
		return
	}

	err = bq.initProcessedFilesTable()
	if err != nil {
		return
	}

	return
}

func (bq *BQLoader) initTable(name string, schema *bigquery.Schema, rangePartition *bigquery.RangePartitioning, timePartition *bigquery.TimePartitioning) error {
	table := bq.dataset.Table(name)
	_, err := table.Metadata(bq.ctx)
	if err == nil {
		return nil
	}
	if !isErrorNotExist(err) {
		return err
	}

	if rangePartition != nil && timePartition != nil {
		return fmt.Errorf("only one of rangePartition and timePartition may be specified")
	}

	metadata := &bigquery.TableMetadata{
		Name:              name,
		RangePartitioning: rangePartition,
		TimePartitioning:  timePartition,
	}
	if schema != nil {
		metadata.Schema = *schema
	}

	err = table.Create(bq.ctx, metadata)
	return err
}

func (bq *BQLoader) initNewOrdersTable() error {
	schema, err := schemas.GetTableSchema(tableOrders, "v1")
	if err != nil {
		return err
	}

	return bq.initTable(tableOrders, schema, nil, nil)
}

func (bq *BQLoader) initOrderFillsTable() error {
	timePartition := &bigquery.TimePartitioning{
		Type:  bigquery.DayPartitioningType,
		Field: "timestamp",
	}

	schema, err := schemas.GetTableSchema(tableFills, "v1")
	if err != nil {
		return err
	}
	return bq.initTable(tableFills, schema, nil, timePartition)
}

func (bq *BQLoader) initTradersTable() error {
	schema, err := schemas.GetTableSchema(tableTraders, "v1")
	if err != nil {
		return err
	}
	return bq.initTable(tableTraders, schema, nil, nil)
}

func (bq *BQLoader) initProcessedFilesTable() error {
	schema, err := schemas.GetTableSchema(tableProcessedFiles, "v1")
	if err != nil {
		return err
	}
	return bq.initTable(tableProcessedFiles, schema, nil, nil)
}

func (bq *BQLoader) initMarketsTable() error {
	schema, err := schemas.GetTableSchema(tableMarkets, "v1")
	if err != nil {
		return err
	}
	return bq.initTable(tableMarkets, schema, nil, nil)
}

func (bq *BQLoader) initTokensTable() error {
	schema, err := schemas.GetTableSchema(tableTokens, "v1")
	if err != nil {
		return err
	}
	return bq.initTable(tableTokens, schema, nil, nil)
}

func isErrorNotExist(err error) bool {
	apiError, ok := err.(*googleapi.Error)
	if !ok {
		return false
	}
	if apiError.Code != 404 {
		return false
	}
	return true
}
