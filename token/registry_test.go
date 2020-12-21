package token

import (
	"testing"

	"github.com/dfuse-io/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Load(t *testing.T) {

	t.Skip("This test will failed till we register BTC meta on mainnet")

	rpcClient := rpc.NewClient("http://api.mainnet-beta.solana.com:80/rpc")

	r := NewRegistry(rpcClient, "ws://api.mainnet-beta.solana.com:80/rpc")
	err := r.Load()
	require.NoError(t, err)

	require.Equal(t, "BTC", r.store["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Meta.Symbol)
	require.Equal(t, uint8(6), r.store["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Decimals)

}
