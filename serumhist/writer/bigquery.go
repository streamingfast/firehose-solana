package writer

import (
	"github.com/dfuse-io/dfuse-solana/serumhist/event"
)

type BigQuery struct {
}

func (b BigQuery) NewOrder(order *event.NewOrder) error {
	panic("implement me")

}
func (b BigQuery) Fill(fill *event.Fill) error {
	panic("implement me")
}

func (b BigQuery) OrderExecuted(executed *event.OrderExecuted) error {
	panic("implement me")
}

func (b BigQuery) OrderClosed(closed *event.OrderClosed) error {
	panic("implement me")
}

func (b BigQuery) OrderCancelled(cancelled *event.OrderCancelled) error {
	panic("implement me")
}
