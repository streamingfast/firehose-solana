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

type Table string

const (
	tableOrders         Table = "orders"
	tableFills          Table = "fills"
	tableTraders        Table = "traders"
	tableProcessedFiles Table = "processed_files"
)

var allTables = []Table{tableOrders, tableFills, tableTraders, tableProcessedFiles}

// TODO: at this point shoudn't this be part of a holistic Table struct?
var codecs = map[Table]*goavro.Codec{}

func (t Table) Exists(ctx context.Context, dataset *bigquery.Dataset) (bool, error) {
	_, err := dataset.Table(t.String()).Metadata(ctx)
	if err != nil {
		if isErrorNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (t Table) Initialize() error {
	specification, err := schemas.GetAvroSchemaV1(t.String())
	if err != nil {
		return fmt.Errorf("unable to retrieve avro schema for table %q: %w", t, err)
	}

	if codecs[t], err = goavro.NewCodec(specification); err != nil {
		return fmt.Errorf("failed to create new codec for table %q: %w", t, err)
	}
	return nil
}

func (t Table) Codec() (*goavro.Codec, error) {
	if c, ok := codecs[t]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("unable to find codec for table %q. Make sure the table was initialized before calling this function", t)
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
