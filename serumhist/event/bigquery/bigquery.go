package bigquery

import (
	"cloud.google.com/go/bigquery"
	"context"
	"encoding/json"
	"fmt"
	"time"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/event"
	"github.com/tidwall/gjson"
)

type FieldMapping struct {
	SourceFieldJsonPath string `json:"path"`
	ExportedFieldName   string `json:"field"`
	ExportedFieldType   string `json:"type"`
}

type Mapping struct {
	FieldMappings []FieldMapping `json:"mappings"`
}

type Row struct {
	mapping *Mapping
	event   interface{}
}

func (r *Row) Save() (row map[string]bigquery.Value, insertID string, err error) {
	eventBytes, err := json.Marshal(r.event)
	if err != nil {
		return nil, "", err
	}

	render := map[string]bigquery.Value{}
	for _, fieldMapping := range r.mapping.FieldMappings {
		switch fieldMapping.ExportedFieldType {
		case "string":
			render[fieldMapping.ExportedFieldName] = gjson.GetBytes(eventBytes, fieldMapping.SourceFieldJsonPath).String()
		case "uint":
			render[fieldMapping.ExportedFieldName] = gjson.GetBytes(eventBytes, fieldMapping.SourceFieldJsonPath).Uint()
		case "int":
			render[fieldMapping.ExportedFieldName] = gjson.GetBytes(eventBytes, fieldMapping.SourceFieldJsonPath).Int()
		case "timestamp":
			seconds := gjson.GetBytes(eventBytes, fieldMapping.SourceFieldJsonPath+".seconds").Int()
			nanos := gjson.GetBytes(eventBytes, fieldMapping.SourceFieldJsonPath+".nanos").Int()
			render[fieldMapping.ExportedFieldName] = time.Unix(seconds, nanos)
		default:
			return nil, "", fmt.Errorf("invalid mapping field type")
		}
	}
	return render, "", nil
}

type BigQuery struct {
	client *bigquery.Client

	orderCreatedMapping *Mapping
	orderCreatedTable   *bigquery.Table
	orderFilledMapping  *Mapping
	orderFilledTable    *bigquery.Table

}

func New() event.Writer {
	return &BigQuery{}
}

func schemaUpdate(ctx context.Context, table *bigquery.Table, schema bigquery.Schema) error {
	tableMetadataToUpdate := bigquery.TableMetadataToUpdate{
		Schema: schema,
	}
	if _, err := table.Update(ctx, tableMetadataToUpdate, ""); err != nil {
		return err
	}
	return nil
}

func (b *BigQuery) NewOrder(ctx context.Context, order *event.NewOrder) error {
	if b.orderCreatedTable == nil {
		return nil
	}

	row := &Row{
		mapping: b.orderCreatedMapping,
		event:   order,
	}
	return b.orderCreatedTable.Inserter().Put(ctx, row)
}

func (b *BigQuery) Fill(ctx context.Context, fill *event.Fill) error {
	if b.orderFilledTable == nil {
		return nil
	}
	return b.orderFilledTable.Inserter().Put(ctx, fill)
}

func (b *BigQuery) OrderExecuted(ctx context.Context, executed *event.OrderExecuted) error {
	return nil
}

func (b *BigQuery) OrderClosed(ctx context.Context, closed *event.OrderClosed) error {
	return nil
}

func (b *BigQuery) OrderCancelled(ctx context.Context, cancelled *event.OrderCancelled) error {
	return nil
}

func (b *BigQuery) WriteCheckpoint(ctx context.Context, checkpoint *pbserumhist.Checkpoint) error {
	panic("implement me")
}

func (b *BigQuery) Checkpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {
	panic("implement me")
}

func (b *BigQuery) Flush(ctx context.Context) (err error) {
	panic("implement me")
}