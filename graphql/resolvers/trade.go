package resolvers

import (
	"encoding/json"
	"fmt"

	"github.com/dfuse-io/solana-go/serum"
)

type SideType string

const (
	SideTypeBid string = "BID"
	SideTypeAsk string = "ASK"
)

type Trade struct {
	inst *serum.Instruction
	err  error
}

func newTrade(inst *serum.Instruction) *Trade {
	return &Trade{
		inst: inst,
	}
}

func (e *Trade) SubscriptionError() error {
	return e.err
}

func (t *Trade) Body() (string, error) {
	cnt, err := json.Marshal(t.inst)
	if err != nil {
		return "", err
	}
	return string(cnt), nil
}
func (t *Trade) Type() string {
	return fmt.Sprintf("%d", t.inst.TypeID)
}

//func (t *Trade) Market() *Market { return newMarket(t.t.Market.Address, t.t.Market.Name) }
//func (t *Trade) Side() string    { return string(t.t.Side) }
//func (t *Trade) Size() float64 {
//	v, _ := t.t.Size.Float64()
//	return v
//}
//func (t *Trade) Price() float64 {
//	v, _ := t.t.Price.Float64()
//	return v
//}
//func (t *Trade) Liquidity() float64 {
//	v, _ := t.t.Liquidity.Float64()
//	return v
//}
//func (t *Trade) Fee() float64 {
//	v, _ := t.t.Liquidity.Float64()
//	return v
//}

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
