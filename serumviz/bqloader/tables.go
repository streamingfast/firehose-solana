package bqloader

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/dfuse-io/dfuse-solana/serumviz/schemas"
	"github.com/linkedin/goavro/v2"
	"google.golang.org/api/googleapi"
)

//TODO: remove table creation logic from here.  will be done in terraform

const (
	tableOrders         = Table("orders")
	tableFills          = Table("fills")
	tableTraders        = Table("traders")
	tableMarkets        = Table("markets")
	tableTokens         = Table("tokens")
	tableProcessedFiles = Table("processed_files")

	schemaVersion = "v1"
)

var allTables = []Table{tableOrders, tableFills, tableTraders, tableMarkets, tableTokens, tableProcessedFiles}

var rangePartitions = map[Table]*bigquery.RangePartitioning{}
var timePartitions = map[Table]*bigquery.TimePartitioning{
	tableFills: &bigquery.TimePartitioning{
		Type:  bigquery.DayPartitioningType,
		Field: "timestamp",
	},
}
var codecs = map[Table]*goavro.Codec{}

type Table string

func (t Table) Exists(ctx context.Context, dataset *bigquery.Dataset) (bool, error) {
	table := dataset.Table(t.String())
	_, err := table.Metadata(ctx)
	if err != nil {
		if isErrorNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (t Table) Create(ctx context.Context, dataset *bigquery.Dataset) error {
	alreadyExists, err := t.Exists(ctx, dataset)
	if err != nil {
		return fmt.Errorf("could not check if table already exists: %w", err)
	}

	if alreadyExists {
		return nil
	}

	rangePartition := rangePartitions[t]
	timePartition := timePartitions[t]

	if rangePartition != nil && timePartition != nil {
		return fmt.Errorf("only one of rangePartition and timePartition may be specified")
	}

	schema, err := t.Schema()
	if err != nil {
		return fmt.Errorf("could not get schema: %w", err)
	}

	metadata := &bigquery.TableMetadata{
		Name:              t.String(),
		RangePartitioning: rangePartition,
		TimePartitioning:  timePartition,
	}
	if schema != nil {
		metadata.Schema = *schema
	}

	table := dataset.Table(string(t))
	err = table.Create(ctx, metadata)
	return err
}

func (t Table) Schema() (*bigquery.Schema, error) {
	return schemas.GetTableSchema(t.String(), schemaVersion)
}

func (t Table) Codec() (*goavro.Codec, error) {
	if c, ok := codecs[t]; ok {
		return c, nil
	}

	specification, err := schemas.GetAvroSpecification(t.String(), schemaVersion)
	if err != nil {
		return nil, err
	}

	c, err := goavro.NewCodec(specification)
	if err != nil {
		return nil, err
	}
	codecs[t] = c
	return c, nil
}

func (t Table) String() string {
	return string(t)
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
