package md

import (
	"testing"

	"gotest.tools/assert"

	"github.com/dfuse-io/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

func Test_loadKnownTokens(t *testing.T) {
	//t.Skip("long running test")

	s := &Server{
		tokenStore:   map[string]*RegisteredToken{},
		tokenListURL: "gs://staging.dfuseio-global.appspot.com/sol-tokens/sol-mainnet-v1.jsonl",
		wsURL:        "ws://api.mainnet-beta.solana.com:80/rpc",
		//rpcClient:    rpc.NewClient("http://api.mainnet-beta.solana.com:80/rpc"),
		rpcClient: rpc.NewClient("https://solana-api.projectserum.com:443"),
	}

	err := s.readKnownTokens()
	require.NoError(t, err)

	assert.Equal(t, 30, len(s.tokenStore))
	assert.Equal(t, "BTC", s.tokenStore["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Meta.Symbol)
}

func Test_loadChainTokens(t *testing.T) {
	t.Skip("This test will failed till we register BTC meta on mainnet")

	rpcClient := rpc.NewClient("http://api.mainnet-beta.solana.com:80/rpc")

	s := &Server{
		rpcClient:  rpcClient,
		wsURL:      "ws://api.mainnet-beta.solana.com:80/rpc",
		tokenStore: map[string]*RegisteredToken{},
	}

	err := s.loadFromChain()
	require.NoError(t, err)

	require.Equal(t, "BTC", s.tokenStore["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Meta.Symbol)
	require.Equal(t, uint8(6), s.tokenStore["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Decimals)

}
