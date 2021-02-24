package resolvers

import "github.com/dfuse-io/dfuse-solana/registry"

// SerumMarket is used to implement both the GraphQL's SerumMarketDailyData and the the SerumMarket.
type SerumMarket struct {
	Address    string
	market     *registry.Market
	baseToken  *registry.Token
	quoteToken *registry.Token

	// For SerumMarketDailyData
	dailyVolumeUSD []DailyVolume
}

func (m SerumMarket) Name() *string {
	if m.market == nil {
		return nil
	}

	if m.market.Name == "" {
		return nil
	}
	return &m.market.Name
}

func (s SerumMarket) BaseToken() *Token {
	if s.baseToken != nil {
		return &Token{s.baseToken}
	}
	return nil
}

func (s SerumMarket) QuoteToken() *Token {
	if s.quoteToken != nil {
		return &Token{s.quoteToken}
	}
	return nil
}

func (s SerumMarket) DailyVolumeUSD() []DailyVolume {
	return s.dailyVolumeUSD
}
