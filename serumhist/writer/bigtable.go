package writer

import (
	"github.com/dfuse-io/dfuse-solana/serumhist/event"
)

type BigTable struct {
}

func (b BigTable) NewOrder(order *event.NewOrder) error {
	panic("implement me")
}

func (b BigTable) Fill(fill *event.Fill) error {
	panic("implement me")
}

func (b BigTable) OrderExecuted(executed *event.OrderExecuted) error {
	panic("implement me")
}

func (b BigTable) OrderClosed(closed *event.OrderClosed) error {
	panic("implement me")
}

func (b BigTable) OrderCancelled(cancelled *event.OrderCancelled) error {
	panic("implement me")
}
