package bqloader

import (
	"github.com/test-go/testify/assert"
	"testing"
)

func Test_FileNameString(t *testing.T) {
	tests := []struct {
		name     string
		input    *FileName
		expected string
	}{
		{
			name:     "basic",
			input:    NewFileName("testPrefix", 0, 100, "a", "b", "2020-01-01-12345"),
			expected: "testPrefix/0-100-a-b-2020-01-01-12345.avro",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.input.String())
		})
	}
}
