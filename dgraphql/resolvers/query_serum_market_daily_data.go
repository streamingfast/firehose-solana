package resolvers

import (
	"fmt"
	"time"

	serumztics "github.com/dfuse-io/dfuse-solana/serumviz/analytics"
	"github.com/dfuse-io/solana-go"
	gqerrs "github.com/graph-gophers/graphql-go/errors"
)

func init() {
	todayFunc = time.Now
}

type SerumMarketDailyDataRequest struct {
	Address string
}

func (r *Root) QuerySerumMarketDailyData(in *SerumMarketDailyDataRequest) (*SerumMarket, error) {
	marketKey, err := solana.PublicKeyFromBase58(in.Address)
	if err != nil {
		return nil, gqerrs.Errorf(`invalid "address" argument %q: %s`, in.Address, err)
	}

	market := r.marketGetter(&marketKey)
	if market == nil {
		return nil, nil
	}

	last24hVolumeUSD, err := r.serumhistAnalytic.TotalVolume(serumztics.Last24Hours())
	if err != nil {
		return nil, fmt.Errorf("unable to retrieved market volume data: %w", err)
	}

	return &SerumMarket{
		Address:    market.Address.String(),
		market:     market,
		baseToken:  r.tokenGetter(&market.BaseToken),
		quoteToken: r.tokenGetter(&market.QuoteToken),

		last24hVolumeUSD: last24hVolumeUSD,
		dailyVolumeUSD:   []DailyVolume{},
	}, nil
}

var todayFunc func() time.Time

func today() time.Time {
	now := todayFunc().UTC()

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
