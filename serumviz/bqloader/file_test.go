package bqloader

import (
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFileNameString(t *testing.T) {
	startBlockID, err := base58.Decode("EdsnkEEWqHwEr4DFBQxNWDuTTkq8MdkiVgsiLKVe9cYQ")
	lastestBlockID, err := base58.Decode("FewYvMSr5w91L5TjayYc3bvG5PN4LgS2oopfNLf9fXZs")
	require.NoError(t, err)
	input := NewFileName("testPrefix", 0, 100, startBlockID, lastestBlockID)
	expected := "testPrefix/0-100-EdsnkEEWqHwEr4DFBQxNWDuTTkq8MdkiVgsiLKVe9cYQ-FewYvMSr5w91L5TjayYc3bvG5PN4LgS2oopfNLf9fXZs"

	assert.Equal(t, expected, input.String())
}
