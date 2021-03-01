package resolvers

import "github.com/dfuse-io/dfuse-solana/registry"

// SerumMarket is used to implement both the GraphQL's SerumMarketDailyData and the the SerumMarket.
type SerumMarket struct {
	Address    string
	market     *registry.Market
	baseToken  *registry.Token
	quoteToken *registry.Token

	// For SerumMarketDailyData
	last24hVolumeUSD float64
	dailyVolumeUSD   []DailyVolume
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

func (s SerumMarket) BaseToken() Token {
	if s.baseToken != nil {
		return Token{nil, s.baseToken}
	}

	return Token{address: &s.market.BaseToken}
}

func (s SerumMarket) QuoteToken() Token {
	if s.quoteToken != nil {
		return Token{nil, s.quoteToken}
	}

	return Token{address: &s.market.QuoteToken}
}

// For SerumMarketDailyData

func (s SerumMarket) Last24hVolumeUSD() Float64 {
	return Float64(s.last24hVolumeUSD)
}

func (s SerumMarket) DailyVolumeUSD() []DailyVolume {
	return s.dailyVolumeUSD
}
