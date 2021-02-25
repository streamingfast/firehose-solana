package schemas

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/bigquery"
)

type Definition struct {
	Fields []DefinitionField `json:"fields"`
}

type DefinitionField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type avroField struct {
	Name string      `json:"name"`
	Type interface{} `json:"type"`
}

type avroLogicalField struct {
	Type        string `json:"type"`
	LogicalType string `json:"logicalType"`
}

func getProjectRootPath() (string, error) {
	project := "dfuse-solana"

	curDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not determine current working directory: %w", err)
	}

	pathParts := []string{"/"}
	pathParts = append(pathParts, strings.Split(curDir, string(filepath.Separator))...)
	for i, elem := range pathParts {
		if elem == project {
			return filepath.Join(pathParts[:i+1]...), nil
		}
	}

	return "", fmt.Errorf("could not determine project root path")
}

func openSchemaFile(filename, version string) ([]byte, error) {
	rootDir, err := getProjectRootPath()
	if err != nil {
		return nil, err
	}
	return ioutil.ReadFile(filepath.Join(rootDir, "serumhist", "bqloader", "schemas", version, fmt.Sprintf("%s.json", filename)))
}

func GetTableSchema(schemaName, version string) (*bigquery.Schema, error) {
	bytes, err := openSchemaFile(schemaName, version)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}

	var schemaDef *Definition
	err = json.Unmarshal(bytes, &schemaDef)
	if err != nil {
		return nil, fmt.Errorf("could not decode schema definition %s: %w", schemaName, err)
	}

	if schemaDef == nil || schemaDef.Fields == nil {
		return nil, fmt.Errorf("no fields are defined for %s", schemaName)
	}

	fieldSchemas := make([]*bigquery.FieldSchema, 0, len(schemaDef.Fields))
	for _, field := range schemaDef.Fields {
		bigQueryType, err := toGoogleType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("error reading field %s in %s: %w", field.Name, schemaName, err)
		}
		fieldSchemas = append(fieldSchemas, &bigquery.FieldSchema{
			Name: field.Name,
			Type: bigQueryType,
		})
	}

	schema := bigquery.Schema(fieldSchemas)
	return &schema, nil
}

func GetAvroSchemaDefinition(schemaName, version string) (string, error) {
	bytes, err := openSchemaFile(schemaName, version)
	if err != nil {
		return "", fmt.Errorf("could not open file: %w", err)
	}

	var schemaDef *Definition
	err = json.Unmarshal(bytes, &schemaDef)
	if err != nil {
		return "", fmt.Errorf("could not decode schema definition %s: %w", schemaName, err)
	}

	if schemaDef == nil || schemaDef.Fields == nil {
		return "", fmt.Errorf("no fields are defined for %s", schemaName)
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
		Fields:    make([]avroField, 0, len(schemaDef.Fields)),
	}

	for _, field := range schemaDef.Fields {
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

func toGoogleType(baseType string) (bigquery.FieldType, error) {
	switch strings.ToLower(baseType) {
	case "int", "int32", "int64", "long":
		return bigquery.IntegerFieldType, nil
	case "string":
		return bigquery.StringFieldType, nil
	case "timestamp":
		return bigquery.TimestampFieldType, nil
	case "bool", "boolean":
		return bigquery.BooleanFieldType, nil
	default:
		return "", fmt.Errorf("unsupported avro type %s", baseType)
	}
}

func toAvroType(baseType string) (interface{}, error) {
	switch strings.ToLower(baseType) {
	case "int", "int32":
		return "int", nil
	case "int64", "long":
		return "long", nil
	case "string":
		return "string", nil
	case "timestamp":
		return avroLogicalField{Type: "long", LogicalType: "timestamp-millis"}, nil
	case "bool", "boolean":
		return "boolean", nil
	default:
		return "", fmt.Errorf("unsupported avro type %s", baseType)
	}
}

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
