package registry

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

func Test_loadKnownTokens(t *testing.T) {
	//t.Skip("long running test")

	s := &Server{
		tokenStore:   map[string]*Token{},
		tokenListURL: "gs://staging.dfuseio-global.appspot.com/sol-tokens/sol-mainnet-v1.jsonl",
		wsURL:        "ws://api.mainnet-beta.solana.com:80/rpc",
		//rpcClient:    rpc.NewClient("http://api.mainnet-beta.solana.com:80/rpc"),
		rpcClient: rpc.NewClient("https://solana-api.projectserum.com:443"),
	}

	err := s.readKnownTokens()
	require.NoError(t, err)

	assert.Equal(t, 30, len(s.tokenStore))
	assert.Equal(t, "BTC", s.tokenStore["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Meta.Symbol)
	assert.Equal(t, "BTC", s.tokenStore["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Meta.Symbol)
}

func Test_loadKnownMarkets(t *testing.T) {
	//t.Skip("long running test")

	s := &Server{
		marketStore:   map[string]*Market{},
		marketListURL: "gs://staging.dfuseio-global.appspot.com/sol-markets/sol-mainnet-v2.jsonl",
	}

	err := s.readKnownMarkets()
	require.NoError(t, err)

	assert.Equal(t, 781, len(s.marketStore))
	assert.Equal(t, &Market{
		Name:       "SOL/USDT",
		Address:    solana.MustPublicKeyFromBase58("7xLk17EQQ5KLDLDe44wCmupJKJjTGd8hs3eSVVhCx932"),
		Deprecated: false,
		ProgramID:  solana.MustPublicKeyFromBase58("EUqojwWA2rd19FZrzeBncJsm38Jm1hEhE3zsmX3bRc2o"),
		BaseToken:  solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112"),
		QuoteToken: solana.MustPublicKeyFromBase58("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4"),
	}, s.marketStore["7xLk17EQQ5KLDLDe44wCmupJKJjTGd8hs3eSVVhCx932"])

	fmt.Println(s.marketStore["7xLk17EQQ5KLDLDe44wCmupJKJjTGd8hs3eSVVhCx932"].BaseToken.String())
}

func Test_loadChainTokens(t *testing.T) {
	t.Skip("This test will failed till we register BTC meta on mainnet")

	rpcClient := rpc.NewClient("http://api.mainnet-beta.solana.com:80/rpc")

	s := &Server{
		rpcClient:  rpcClient,
		wsURL:      "ws://api.mainnet-beta.solana.com:80/rpc",
		tokenStore: map[string]*Token{},
	}

	err := s.loadFromChain()
	require.NoError(t, err)

	require.Equal(t, "BTC", s.tokenStore["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Meta.Symbol)
	require.Equal(t, uint8(6), s.tokenStore["9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"].Decimals)

}
