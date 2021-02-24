package resolvers

import (
	"testing"
	"time"

	"github.com/dfuse-io/dfuse-solana/registry"
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
		in          *SerumMarketDailyDataRequest
		expected    *SerumMarket
		expectedErr error
	}{
		{
			"found with daily volume",
			&preState{markets: markets, tokens: tokens},
			1456666.01,
			&SerumMarketDailyDataRequest{Address: "Gw78CYLLFbgmmn4rps9KoPAnNtBQ2S1foL2Mn6Z5ZHYB"},
			toSerumMarket(markets[0], tokens, func(m *SerumMarket) {
				m.dailyVolumeUSD = []DailyVolume{{date: tTime(t, "2021-02-22T00:00:00Z"), value: 1456666.01}}
			}),
			nil,
		},
		{
			"not found market",
			&preState{markets: markets, tokens: tokens},
			0,
			&SerumMarketDailyDataRequest{Address: "Gf78CYLLFbgmmn4rps9KoPAnNtBQ2S1foL2Mn6Z5ZHYB"},
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
				marketGetter:      newMarketGetter(test.state.markets),
				tokenGetter:       newTokenGetter(test.state.tokens),
				serumhistAnalytic: &serumTestAnalyzable{test.volume},
			}

			actual, err := root.QuerySerumMarketDailyData(test.in)
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
