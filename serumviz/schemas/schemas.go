package schemas

import (
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	rice "github.com/GeertJohan/go.rice"
)

//go:generate rice embed-go

func GetBQSchemaV1(name string) (*bigquery.Schema, error) {
	box := rice.MustFindBox("v1")
	cnt, err := box.Bytes(fmt.Sprintf("%s.json", name))
	if err != nil {
		return nil, fmt.Errorf("unable to get schema %q: %w", name, err)
	}

	schema := bigquery.Schema{}
	err = json.Unmarshal([]byte(cnt), &schema)
	if err != nil {
		return nil, fmt.Errorf("unable to parse schema: %w", err)
	}

	return &schema, nil
}

func GetAvroSchemaV1(schemaName string) (string, error) {
	box := rice.MustFindBox("v1")
	cnt, err := box.Bytes(fmt.Sprintf("%s.json", schemaName))
	if err != nil {
		return "", fmt.Errorf("unable to get schema %q: %w", schemaName, err)
	}

	bqSchema := bigquery.Schema{}
	err = json.Unmarshal([]byte(cnt), &bqSchema)
	if err != nil {
		return "", fmt.Errorf("unable to parse schema: %w", err)
	}

	type codec struct {
		Namespace string      `json:"namespace"`
		Type      string      `json:"type"`
		Name      string      `json:"name"`
		Fields    []avroField `json:"fields"`
	}

	result := &codec{
		Namespace: "io.dfuse",
		Type:      "record",
		Name:      toCamelCase(schemaName),
		Fields:    make([]avroField, 0, len(bqSchema)),
	}

	for _, field := range bqSchema {
		avroType, err := toAvroType(field.Type)
		if err != nil {
			return "", fmt.Errorf("error reading field %s in %s: %w", field.Name, schemaName, err)
		}
		result.Fields = append(result.Fields, avroField{
			Name: field.Name,
			Type: avroType,
		})
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultBytes), nil
}

type avroField struct {
	Name string      `json:"name"`
	Type interface{} `json:"type"`
}

type avroLogicalField struct {
	Type        string `json:"type"`
	LogicalType string `json:"logicalType"`
}

func toAvroType(bgFieldType bigquery.FieldType) (interface{}, error) {
	switch bgFieldType {
	case bigquery.StringFieldType:
		return "string", nil
	case bigquery.IntegerFieldType:
		return "long", nil
	case bigquery.FloatFieldType:
		panic("un")
	case bigquery.BooleanFieldType:
		return "boolean", nil
	case bigquery.TimestampFieldType:
		return avroLogicalField{Type: "long", LogicalType: "timestamp-millis"}, nil
	// TODO: add support as the need arises
	//case bigquery.BytesFieldType:
	//case bigquery.RecordFieldType:
	//case bigquery.DateFieldType:
	//case bigquery.TimeFieldType:
	//case bigquery.DateTimeFieldType:
	//case bigquery.NumericFieldType:
	//case bigquery.GeographyFieldType:
	default:
		return "", fmt.Errorf("unsupported avro type %q", bgFieldType)
	}
}

// TODO: should we use package "github.com/iancoleman/strcase"??
func toCamelCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	uppercaseAcronym := map[string]string{
		"ID": "id",
	}

	if a, ok := uppercaseAcronym[s]; ok {
		s = a
	}

	n := strings.Builder{}
	n.Grow(len(s))
	capNext := true
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if capNext {
			if vIsLow {
				v += 'A'
				v -= 'a'
			}
		} else if i == 0 {
			if vIsCap {
				v += 'a'
				v -= 'A'
			}
		}
		if vIsCap || vIsLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
		}
	}
	return n.String()
}
