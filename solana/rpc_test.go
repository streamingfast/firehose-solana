package solana

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRPCClient_GetConfirmedBlock(t *testing.T) {

	//rpcClient := NewRPCClient("api.mainnet-beta.solana.com:443")
	rpcClient := NewRPCClient("testnet.solana.com:8899")
	err := rpcClient.GetConfirmedBlock(46243868)
	require.NoError(t, err)
}
