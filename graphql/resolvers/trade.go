package resolvers

import "github.com/dfuse-io/dfuse-solana/graphql"

type SideType string

const (
	SideTypeBid string = "BID"
	SideTypeAsk string = "ASK"
)

type Trade struct {
	t   *graphql.Trade
	err error
}

func newTrade(t *graphql.Trade) *Trade {
	return &Trade{
		t: t,
	}
}

func (e *Trade) SubscriptionError() error {
	return e.err
}

func (t *Trade) Market() *Market { return newMarket(t.t.Market.Address, t.t.Market.Name) }
func (t *Trade) Side() string    { return string(t.t.Side) }
func (t *Trade) Size() float64 {
	v, _ := t.t.Size.Float64()
	return v
}
func (t *Trade) Price() float64 {
	v, _ := t.t.Price.Float64()
	return v
}
func (t *Trade) Liquidity() float64 {
	v, _ := t.t.Liquidity.Float64()
	return v
}
func (t *Trade) Fee() float64 {
	v, _ := t.t.Liquidity.Float64()
	return v
}

type Market struct {
	Name    *string
	Address string
}

func newMarket(address, name string) *Market {
	var a *string
	if name != "" {
		a = &name
	}
	return &Market{
		Name:    a,
		Address: address,
	}
}
