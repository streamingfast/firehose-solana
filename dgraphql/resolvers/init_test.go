package resolvers

import (
	"time"

	serumanalytics "github.com/dfuse-io/dfuse-solana/serumhist/analytics"
	"github.com/dfuse-io/solana-go"
)

var pubKey = solana.MustPublicKeyFromBase58

type serumTestAnalyzable struct {
	volume24h float64
}

func (s *serumTestAnalyzable) GetHourlyFillsVolume(dateRange *serumanalytics.DateRange, key *solana.PublicKey) ([]*serumanalytics.FillVolume, error) {
	return []*serumanalytics.FillVolume{
		{
			USDVolume:         "9618.624516786102",
			Timestamp:         time.Now(),
			SlotNum:           123,
			TrxIdx:            2,
			InstIdx:           4,
			MarketAddress:     "H3APNWA8bZW2gLMSq5sRL41JSMmEJ648AqoEdDgLcdvB",
			BaseTokenAddress:  "",
			QuoteTokenAddress: "",
		},
	}, nil
}

func (s *serumTestAnalyzable) Get24hVolume() (float64, error) {
	return s.volume24h, nil
}
