package bqloader

import (
	"github.com/test-go/testify/assert"
	"testing"
)

func Test_FileNameString(t *testing.T) {
	input := NewFileName("testPrefix", 0, 100, "a", "b", "2020-01-01-12345")
	expected := "testPrefix/0-100-a-b-2020-01-01-12345.avro"

	assert.Equal(t, expected, input.String())

}

func Test_parseLatestInfoFromFilename(t *testing.T) {
	input := "testPrefix/0-100-a-b-2020-01-01-12345.avro"
	expected := &FileName{
	LatestSlotNum: 100,
	LatestSlotId:  "b",
}

	output, err := parseLatestInfoFromFilename(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, output)
}
