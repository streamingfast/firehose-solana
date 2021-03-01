package resolvers

import (
	"fmt"

	"github.com/dfuse-io/dfuse-solana/registry"
	serumztics "github.com/dfuse-io/dfuse-solana/serumviz/analytics"
)

type SerumMarket struct {
	Address    string
	market     *registry.Market
	baseToken  *registry.Token
	quoteToken *registry.Token

	dailyVolumeUSD []DailyVolume

	serumhistAnalyzable serumztics.Analyzable
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

func (s SerumMarket) Last24HoursVolumeUSD() (Float64, error) {
	return s.lastVolumeData("24 hours", serumztics.Last24Hours())
}

func (s SerumMarket) Last7DaysVolumeUSD() (Float64, error) {
	return s.lastVolumeData("7 days", serumztics.Last7Days())
}

func (s SerumMarket) Last30DaysVolumeUSD() (Float64, error) {
	return s.lastVolumeData("30 days", serumztics.Last30Days())
}

func (s SerumMarket) lastVolumeData(tag string, dateRange serumztics.DateRange) (Float64, error) {
	volume, err := s.serumhistAnalyzable.TotalVolume(dateRange)
	if err != nil {
		return Float64(0), fmt.Errorf("unable to retrieved last %s market volume: %w", tag, err)
	}

	return Float64(volume), nil
}

func (s SerumMarket) DailyVolumeUSD() []DailyVolume {
	return s.dailyVolumeUSD
}
