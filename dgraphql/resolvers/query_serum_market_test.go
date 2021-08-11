package resolvers

import (
	"testing"
	"time"

	"github.com/streamingfast/sf-solana/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuerySerumMarketDailyData(t *testing.T) {
	type preState struct {
		markets []*registry.Market
		tokens  []*registry.Token
	}

	markets := []*registry.Market{
		{Address: pubKey("Gw78CYLLFbgmmn4rps9KoPAnNtBQ2S1foL2Mn6Z5ZHYB"), Name: "SXP/USDT", BaseToken: pubKey("BXXkv6z8ykpG1yuvUDPgh732wzVHB69RnB9YgSYh3itW"), QuoteToken: pubKey("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4")},
	}

	tokens := []*registry.Token{
		{Address: pubKey("BQcdHdAQW1hczDbBi9hiegXAR7A98Q9jx3X3iBBBDiq4"), Meta: &registry.TokenMeta{Name: "Wrapped USDT", Symbol: "USDT"}, Decimals: 6},
		{Address: pubKey("BXXkv6z8ykpG1yuvUDPgh732wzVHB69RnB9YgSYh3itW"), Meta: &registry.TokenMeta{Name: "SXP", Symbol: "SXP"}, Decimals: 9},
	}

	tests := []struct {
		name        string
		state       *preState
		volume      float64
		in          *SerumMarketRequest
		expected    *SerumMarket
		expectedErr error
	}{
		// FIXME: Those tests are bad, they used our internal GraphQL struct which sometimes does on-the-fly fetching
		//        of data. What this means is that checking the actual content is impossible.
		//
		//        The real thing that we need is a "GraphQL test executor". It runs a specific operation as a GraphQL document
		//        with variables and everything and check that the JSON output is what we expect.
		// {
		// 	"found with daily volume",
		// 	&preState{markets: markets, tokens: tokens},
		// 	1456666.01,
		// 	&SerumMarketRequest{Address: "Gw78CYLLFbgmmn4rps9KoPAnNtBQ2S1foL2Mn6Z5ZHYB"},
		// 	toSerumMarket(markets[0], tokens, func(m *SerumMarket) {
		// 		m.last = 1456666.01
		// 		m.dailyVolumeUSD = []DailyVolume{}
		// 	}),
		// 	nil,
		// },
		{
			"not found market",
			&preState{markets: markets, tokens: tokens},
			0,
			&SerumMarketRequest{Address: "Gf78CYLLFbgmmn4rps9KoPAnNtBQ2S1foL2Mn6Z5ZHYB"},
			nil,
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			todayFunc = func() time.Time {
				return tTime(t, "2021-02-22T00:00:00Z")
			}
			root := &Root{
				marketGetter:        newMarketGetter(test.state.markets),
				tokenGetter:         newTokenGetter(test.state.tokens),
				serumhistAnalyzable: &serumTestAnalyzable{test.volume},
			}

			actual, err := root.QuerySerumMarket(test.in)
			if test.expectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			} else {
				assert.Equal(t, test.expectedErr, err)
			}
		})
	}
}

func tTime(t *testing.T, date string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, date)
	require.NoError(t, err)

	return parsed
}
