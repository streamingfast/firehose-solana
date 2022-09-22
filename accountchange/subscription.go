package accountchange

import (
	"reflect"

	"github.com/streamingfast/solana-go"
)

type Subscription struct {
	account           solana.PublicKey
	subID             uint64
	stream            chan *Result
	err               chan error
	reflectType       reflect.Type
	closeFunc         func(err error)
	unsubscribeMethod string
}

func newSubscription(account solana.PublicKey, closeFunc func(err error)) *Subscription {

	//todo: register to the reader backed account data stream.

	return &Subscription{
		account:   account,
		stream:    make(chan *Result, 200),
		err:       make(chan error, 1),
		closeFunc: closeFunc,
	}
}

type Result struct {
	Context *ResultContext `json:"context"`
	Value   *ResultValue   `json:"value"`
}

type ResultContext struct {
	Slot uint64 `json:"slot"`
}

type ResultValue struct {
	Data solana.Data `json:"data"`
	//todo: not sure we will get those value from reader...
	//Executable bool             `json:"executable"`
	//Lamports   uint64           `json:"lamports"`
	//Owner      solana.PublicKey `json:"owner"`
	//RentEpoch  uint64           `json:"rentEpoch"`
}

func (s *Subscription) Recv() (*Result, error) {
	select {
	case d := <-s.stream:
		return d, nil
	case err := <-s.err:
		return nil, err
	}
}

func (s *Subscription) Unsubscribe() {
	//todo: unregister to the reader backed account data stream.
	s.unsubscribe(nil)
}

func (s *Subscription) unsubscribe(err error) {
	s.closeFunc(err)

}
