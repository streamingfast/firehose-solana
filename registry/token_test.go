package registry

import (
	"context"
	"fmt"
	"testing"

	"github.com/streamingfast/solana-go"
	solrpc "github.com/streamingfast/solana-go/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToken_Display(t *testing.T) {
	token := &Token{
		Address:  solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112"),
		Decimals: 9,
		Meta: &TokenMeta{
			Symbol: "SOL",
		},
	}

	assert.Equal(t, "2.283130 SOL", token.Display(2283130000))

	token = &Token{
		Address:  solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112"),
		Decimals: 9,
	}
	assert.Equal(t, "2.283130 So11..1112", token.Display(2283130000))

}

func Test_loadKnownTokens(t *testing.T) {
	t.Skip("long running test")

	tokensListURL := "gs://staging.dfuseio-global.appspot.com/sol-tokens/sol-mainnet-v1.jsonl"

	tokens, err := ReadKnownTokens(context.Background(), tokensListURL)
	require.NoError(t, err)

	assert.Equal(t, 30, len(tokens))
	assert.Equal(t, &Token{
		Address:               solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112"),
		MintAuthorityOption:   0,
		MintAuthority:         solana.PublicKey{},
		Supply:                0,
		Decimals:              9,
		IsInitialized:         true,
		FreezeAuthorityOption: 0,
		FreezeAuthority:       solana.PublicKey{},
		Meta: &TokenMeta{
			Logo:    "",
			Name:    "Solana",
			Symbol:  "SOL",
			Website: "",
		},
	}, tokens["So11111111111111111111111111111111111111112"])
}

func Test_syncKnownTokens(t *testing.T) {
	//t.Skip("long running test")

	tokens := []*Token{
		{Address: solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112"), Meta: &TokenMeta{Symbol: "SOL"}},
		//{Address: solana.MustPublicKeyFromBase58("9n4nbM75f5Ui33ZbPYXn59EwSgE8CGsHtAeTH5YFeJ9E"), Meta: &TokenMeta{Symbol: "BTC"}},
		//{Address: solana.MustPublicKeyFromBase58("2FPyTwcZLUg1MDrwsyoP4D6s1tM7hAkHYRjkNb5w6Pxk"), Meta: &TokenMeta{Symbol: "ETH"}},
		//{Address: solana.MustPublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"), Meta: &TokenMeta{Symbol: "USDC"}},
		//{Address: solana.MustPublicKeyFromBase58("3JSf5tPeuscJGtaCp5giEiDhv51gQ4v3zWg8DGgyLfAB"), Meta: &TokenMeta{Symbol: "YFI"}},
	}
	rpc := solrpc.NewClient("https://solana-api.projectserum.com")

	out, err := SyncKnownTokens(rpc, tokens)
	require.NoError(t, err)
	fmt.Println(out[0].FreezeAuthority.String())
}
