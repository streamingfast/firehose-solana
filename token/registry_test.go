package token

import (
	"testing"

	"github.com/dfuse-io/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

func TestRegistry_loadName(t *testing.T) {

	r := NewRegistry(nil, "")
	err := r.loadNames()
	require.NoError(t, err)

	require.Equal(t, "BTC", r.metas["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Symbol)

}

func TestRegistry_Load(t *testing.T) {

	rpcClient := rpc.NewClient("http://api.mainnet-beta.solana.com:80/rpc")

	r := NewRegistry(rpcClient, "ws://api.mainnet-beta.solana.com:80/rpc")
	err := r.Load()
	require.NoError(t, err)

	require.Equal(t, "BTC", r.store["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Meta.Symbol)
	require.Equal(t, uint8(6), r.store["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Decimals)

}
