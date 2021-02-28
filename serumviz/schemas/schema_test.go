package schemas

import (
	"testing"

	"github.com/test-go/testify/require"

	"github.com/davecgh/go-spew/spew"
)

func Test_getSchema(t *testing.T) {
	temp, err := GetBQSchemaV1("fills")
	require.NoError(t, err)
	spew.Dump(temp)
}

func Test_GetAvroSpecification(t *testing.T) {
	temp, err := GetAvroSchemaV1("fills")
	require.NoError(t, err)
	spew.Dump(temp)
}
