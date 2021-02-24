package bqloader

import (
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
)

//TODO: define table partitions

func (bq *BQLoader) InitTables() (err error) {
	err = bq.initNewOrdersTable()
	if err != nil {
		return
	}

	err = bq.initOrderFillsTable()
	if err != nil {
		return
	}

	err = bq.initTradingAccountTable()
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
	return bq.initTable(newOrder, nil, nil, nil)
}

func (bq *BQLoader) initOrderFillsTable() error {
	timePartition := &bigquery.TimePartitioning{
		Type:  bigquery.DayPartitioningType,
		Field: "timestamp",
	}
	return bq.initTable(fillOrder, nil, nil, timePartition)
}

func (bq *BQLoader) initTradingAccountTable() error {
	schema := bigquery.Schema{
		&bigquery.FieldSchema{Name: "account", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "trader", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "slot_num", Type: bigquery.IntegerFieldType},
	}
	return bq.initTable(tradingAccount, &schema, nil, nil)
}

func (bq *BQLoader) initProcessedFilesTable() error {
	schema := bigquery.Schema{
		&bigquery.FieldSchema{Name: "table", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "file", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "timestamp", Type: bigquery.TimestampFieldType},
	}
	return bq.initTable(processedFiles, &schema, nil, nil)
}

func (bq *BQLoader) initMarketsTable() error {
	schema := bigquery.Schema{
		&bigquery.FieldSchema{Name: "name", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "address", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "deprecated", Type: bigquery.BooleanFieldType},
		&bigquery.FieldSchema{Name: "program_id", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "base_token", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "quote_token", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "base_lot_size", Type: bigquery.IntegerFieldType},
		&bigquery.FieldSchema{Name: "quote_lot_size", Type: bigquery.IntegerFieldType},
		&bigquery.FieldSchema{Name: "request_queue", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "event_queue", Type: bigquery.StringFieldType},
	}
	return bq.initTable(markets, &schema, nil, nil)
}

func (bq *BQLoader) initTokensTable() error {
	schema := bigquery.Schema{
		&bigquery.FieldSchema{Name: "name", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "symbol", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "address", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "mint_authority_option", Type: bigquery.IntegerFieldType},
		&bigquery.FieldSchema{Name: "mint_authority", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "supply", Type: bigquery.IntegerFieldType},
		&bigquery.FieldSchema{Name: "decimals", Type: bigquery.IntegerFieldType},
		&bigquery.FieldSchema{Name: "is_initialized", Type: bigquery.BooleanFieldType},
		&bigquery.FieldSchema{Name: "freeze_authority_option", Type: bigquery.IntegerFieldType},
		&bigquery.FieldSchema{Name: "freeze_authority", Type: bigquery.StringFieldType},
		&bigquery.FieldSchema{Name: "verified", Type: bigquery.BooleanFieldType},
	}
	return bq.initTable(tokens, &schema, nil, nil)
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
