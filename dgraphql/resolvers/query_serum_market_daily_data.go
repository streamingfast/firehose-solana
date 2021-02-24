package resolvers

import (
	"time"

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

	return &SerumMarket{
		Address:    market.Address.String(),
		market:     market,
		baseToken:  r.tokenGetter(&market.BaseToken),
		quoteToken: r.tokenGetter(&market.QuoteToken),

		dailyVolumeUSD: []DailyVolume{{date: today(), value: 1456666.01}},
	}, nil
}

var todayFunc func() time.Time

func today() time.Time {
	now := todayFunc().UTC()

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
