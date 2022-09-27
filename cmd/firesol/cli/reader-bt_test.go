package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/streamingfast/dstore"
	"github.com/stretchr/testify/require"
)

func Test_findStartEndBlock(t *testing.T) {
	store, err := dstore.NewDBinStore("gs://dfuseio-global-blocks-us/sol-mainnet/v1")
	require.NoError(t, err)
	start, stop := findStartEndBlock(context.Background(), 0, 0, store)
	fmt.Println(start)
	fmt.Println(stop)
}
