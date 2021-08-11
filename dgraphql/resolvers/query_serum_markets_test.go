package resolvers

import (
	"bytes"
	"testing"

	"github.com/dfuse-io/dfuse-solana/registry"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/dgraphql/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuerySerumMarkets(t *testing.T) {
	type preState struct {
		markets []*registry.Market
		tokens  []*registry.Token
	}

	markets := []*registry.Market{
		// Be aware when mapping to test cases above that the markets are sorted later on, so order does not match here
		{Address: pubKey("Gw78CYLLFbgmmn4rps9KoPAnNtBQ2S1foL2Mn6Z5ZHYB"), Name: "MNO/THE", BaseToken: pubKey("BXXkv6z8ykpG1yuvUDPgh732wzVHB69RnB9YgSYh3itW"), QuoteToken: pubKey("BXXkv6z8ykpG1yuvUDPgh732wzVHB69RnB9YgSYh3itW")},
		{Address: pubKey("97NiXHUNkpYd1eb2HthSDGhaPfepuqMAV3QsZhAgb1wm"), Name: "", BaseToken: pubKey("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4"), QuoteToken: pubKey("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4")},
		{Address: pubKey("4QL5AQvXdMSCVZmnKXiuMMU83Kq3LCwVfU8CyznqZELG"), Name: "ABC/HGF", BaseToken: pubKey("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"), QuoteToken: pubKey("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")},
		{Address: pubKey("GpdYLFbKHeSeDGqsnQ4jnP7D1294iBpQcsN1VPwhoaFS"), Name: "UNI/WUSDC", BaseToken: pubKey("BXXkv6z8ykpG1yuvUDPgh732wzVHB69RnB9YgSYh3itW"), QuoteToken: pubKey("BXXkv6z8ykpG1yuvUDPgh732wzVHB69RnB9YgSYh3itW")},
		{Address: pubKey("MNiXHUNkpYd1eb2HthSDGhaPfepuqMAV3QsZhAgb1wm"), Name: "", BaseToken: pubKey("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4"), QuoteToken: pubKey("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4")},
		{Address: pubKey("ZNiXHUNkpYd1eb2HthSDGhaPfepuqMAV3QsZhAgb1wm"), Name: "", BaseToken: pubKey("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4"), QuoteToken: pubKey("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4")},
	}

	tokens := []*registry.Token{
		{Address: pubKey("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4"), Meta: &registry.TokenMeta{Name: "Wrapped USDT", Symbol: "USDT"}, Decimals: 6},
		{Address: pubKey("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"), Meta: &registry.TokenMeta{Name: "The SRM", Symbol: "SRM"}, Decimals: 8},
		{Address: pubKey("BXXkv6z8ykpG1yuvUDPgh732wzVHB69RnB9YgSYh3itW"), Meta: &registry.TokenMeta{Name: "SXP", Symbol: "SXP"}, Decimals: 9},
	}

	tests := []struct {
		name        string
		state       *preState
		in          *SerumMarketsRequest
		expected    *SerumMarketConnection
		expectedErr error
	}{
		{
			"from scratch limited",
			&preState{markets: markets, tokens: tokens},
			&SerumMarketsRequest{Count: newCount(3)},
			&SerumMarketConnection{
				TotalCount: 6,
				Edges: []*SerumMarketEdge{
					NewSerumMarketEdge(toSerumMarket(markets[2], tokens, nil), markets[2].Address.String()),
					NewSerumMarketEdge(toSerumMarket(markets[0], tokens, nil), markets[0].Address.String()),
					NewSerumMarketEdge(toSerumMarket(markets[3], tokens, nil), markets[3].Address.String()),
				},
				PageInfo: NewPageInfo("4QL5AQvXdMSCVZmnKXiuMMU83Kq3LCwVfU8CyznqZELG", "GpdYLFbKHeSeDGqsnQ4jnP7D1294iBpQcsN1VPwhoaFS", true),
			},
			nil,
		},
		{
			"from cursor limited",
			&preState{markets: markets, tokens: tokens},
			&SerumMarketsRequest{Count: newCount(2), Cursor: newCursor("GpdYLFbKHeSeDGqsnQ4jnP7D1294iBpQcsN1VPwhoaFS")},
			&SerumMarketConnection{
				TotalCount: 6,
				Edges: []*SerumMarketEdge{
					NewSerumMarketEdge(toSerumMarket(markets[4], tokens, nil), markets[4].Address.String()),
					NewSerumMarketEdge(toSerumMarket(markets[5], tokens, nil), markets[5].Address.String()),
				},
				PageInfo: NewPageInfo("MNiXHUNkpYd1eb2HthSDGhaPfepuqMAV3QsZhAgb1wm", "ZNiXHUNkpYd1eb2HthSDGhaPfepuqMAV3QsZhAgb1wm", true),
			},
			nil,
		},
		{
			"from cursor all",
			&preState{markets: markets, tokens: tokens},
			&SerumMarketsRequest{Count: newCount(6), Cursor: newCursor("GpdYLFbKHeSeDGqsnQ4jnP7D1294iBpQcsN1VPwhoaFS")},
			&SerumMarketConnection{
				TotalCount: 6,
				Edges: []*SerumMarketEdge{
					NewSerumMarketEdge(toSerumMarket(markets[4], tokens, nil), markets[4].Address.String()),
					NewSerumMarketEdge(toSerumMarket(markets[5], tokens, nil), markets[5].Address.String()),
					NewSerumMarketEdge(toSerumMarket(markets[1], tokens, nil), markets[1].Address.String()),
				},
				PageInfo: NewPageInfo("MNiXHUNkpYd1eb2HthSDGhaPfepuqMAV3QsZhAgb1wm", "97NiXHUNkpYd1eb2HthSDGhaPfepuqMAV3QsZhAgb1wm", false),
			},
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := &Root{
				marketsGetter: func() []*registry.Market { return test.state.markets },
				tokenGetter:   newTokenGetter(test.state.tokens),
			}

			actual, err := root.QuerySerumMarkets(test.in)
			if test.expectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			} else {
				assert.Equal(t, test.expectedErr, err)
			}
		})
	}
}

func toSerumMarket(market *registry.Market, tokens []*registry.Token, customizer func(m *SerumMarket)) *SerumMarket {
	tokenGetter := newTokenGetter(tokens)

	out := &SerumMarket{
		Address:    market.Address.String(),
		market:     market,
		baseToken:  tokenGetter(&market.BaseToken),
		quoteToken: tokenGetter(&market.QuoteToken),
	}
	if customizer != nil {
		customizer(out)
	}

	return out
}

func newMarketGetter(markets []*registry.Market) func(in *solana.PublicKey) *registry.Market {
	return func(in *solana.PublicKey) *registry.Market {
		for _, market := range markets {
			if bytes.Equal(market.Address[:], (*in)[:]) {
				return market
			}
		}
		return nil
	}
}

func newTokenGetter(tokens []*registry.Token) func(in *solana.PublicKey) *registry.Token {
	return func(in *solana.PublicKey) *registry.Token {
		for _, token := range tokens {
			if bytes.Equal(token.Address[:], (*in)[:]) {
				return token
			}
		}
		return nil
	}
}

func newCount(in uint64) *types.Uint64 {
	if in == 0 {
		return nil
	}

	value := types.Uint64(in)
	return &value
}

func newCursor(in string) *string {
	return &in
}
