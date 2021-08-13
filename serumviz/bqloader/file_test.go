package bqloader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileNameString(t *testing.T) {
	input := NewFileName("testPrefix", 0, 100, "EdsnkEEWqHwEr4DFBQxNWDuTTkq8MdkiVgsiLKVe9cYQ", "FewYvMSr5w91L5TjayYc3bvG5PN4LgS2oopfNLf9fXZs")
	expected := "testPrefix/0-100-EdsnkEEWqHwEr4DFBQxNWDuTTkq8MdkiVgsiLKVe9cYQ-FewYvMSr5w91L5TjayYc3bvG5PN4LgS2oopfNLf9fXZs"

	assert.Equal(t, expected, input.String())

}
