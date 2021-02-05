package snapshot

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnCompress(t *testing.T) {
	f, err := os.Open("/Users/cbillett/t/toto.tar.bz2")
	require.NoError(t, err)
	err = unCompress(f, "/tmp")
	require.NoError(t, err)
}
