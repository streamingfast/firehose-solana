package registry

import (
	"context"
	"testing"

	"github.com/streamingfast/solana-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ReadKnownMarkets(t *testing.T) {
	t.Skip("long running test")

	marketListURL := "gs://staging.dfuseio-global.appspot.com/sol-markets/sol-mainnet-v1.jsonl"

	markets, err := ReadKnownMarkets(context.Background(), marketListURL)
	require.NoError(t, err)

	assert.Equal(t, 781, len(markets))
	assert.Equal(t, &Market{
		Name:         "SOL/USDT",
		Address:      solana.MustPublicKeyFromBase58("7xLk17EQQ5KLDLDe44wCmupJKJjTGd8hs3eSVVhCx932"),
		Deprecated:   false,
		ProgramID:    solana.MustPublicKeyFromBase58("EUqojwWA2rd19FZrzeBncJsm38Jm1hEhE3zsmX3bRc2o"),
		BaseToken:    solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112"),
		QuoteToken:   solana.MustPublicKeyFromBase58("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4"),
		BaseLotSize:  100000000,
		QuoteLotSize: 100,
		RequestQueue: solana.MustPublicKeyFromBase58("G1GrHZfpAkH6j5FrZWCi76T34z1naR8mT8MrfVoDzoQh"),
		EventQueue:   solana.MustPublicKeyFromBase58("9Yzx2KU2MLa3goSc9PxgUvj4Lw5nN6fQG8fekKuPGkoN"),
	}, markets["7xLk17EQQ5KLDLDe44wCmupJKJjTGd8hs3eSVVhCx932"])
}
