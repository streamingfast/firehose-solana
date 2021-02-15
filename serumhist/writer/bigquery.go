package writer

import (
	"github.com/dfuse-io/dfuse-solana/serumhist/event"
)

type Bigquery struct {
}

func (b Bigquery) NewOrder(order *event.NewOrder) error {
	panic("implement me")

}
func (b Bigquery) Fill(fill *event.Fill) error {
	panic("implement me")
}

func (b Bigquery) OrderExecuted(executed *event.OrderExecuted) error {
	panic("implement me")
}

func (b Bigquery) OrderClosed(closed *event.OrderClosed) error {
	panic("implement me")
}

func (b Bigquery) OrderCancelled(cancelled *event.OrderCancelled) error {
	panic("implement me")
}
