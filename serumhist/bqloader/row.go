package bqloader

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
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

func schemaUpdate(ctx context.Context, table *bigquery.Table, schema bigquery.Schema) error {
	tableMetadataToUpdate := bigquery.TableMetadataToUpdate{
		Schema: schema,
	}
	if _, err := table.Update(ctx, tableMetadataToUpdate, ""); err != nil {
		return err
	}
	return nil
}
