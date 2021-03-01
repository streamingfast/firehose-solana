package resolvers

import (
	"time"

	serumanalytics "github.com/dfuse-io/dfuse-solana/serumviz/analytics"
	"github.com/dfuse-io/solana-go"
)

var pubKey = solana.MustPublicKeyFromBase58

type serumTestAnalyzable struct {
	volume24h float64
}

func (s *serumTestAnalyzable) TotalVolume(dateRange serumanalytics.DateRange) (float64, error) {
	return 1456666.01, nil
}

func (s *serumTestAnalyzable) FillsVolume(dateRange *serumanalytics.DateRange, granularity *serumanalytics.Granularity, key *solana.PublicKey) ([]*serumanalytics.FillVolume, error) {
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
