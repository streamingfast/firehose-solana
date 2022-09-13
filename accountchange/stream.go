package accountchange

import (
	"context"
	"fmt"

	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/rpc"
	"github.com/streamingfast/solana-go/rpc/ws"
)

type Stream struct {
	wsClient *ws.Client
}

func NewStream(wsClient *ws.Client) Stream {
	return Stream{
		wsClient: wsClient,
	}
}

func (s *Stream) WatchAccount(ctx context.Context, account solana.PublicKey) (*Subscription, error) {
	//todo: Replace ws subscription by a reader backed account data stream.
	wsSub, err := s.wsClient.AccountSubscribe(account, rpc.CommitmentRecent)
	if err != nil {
		return nil, fmt.Errorf("watch account: ws sub: %w", err)
	}

	//todo: move this in the reader backed account data stream.
	sub := newSubscription(account, nil)
	for {
		wsRes, err := wsSub.Recv(ctx)
		if err != nil {
			sub.err <- err
		}

		wsAccountResult := wsRes.(*ws.AccountResult)
		wsAcc := wsAccountResult.Value.Account

		sub.stream <- &Result{
			Context: &ResultContext{
				Slot: wsAccountResult.Context.Slot,
			},
			Value: &ResultValue{
				Data: wsAcc.Data,
				//todo: not sure we will get those value from reader...
				//Executable: wsAcc.Executable,
				//Lamports:   uint64(wsAcc.Lamports),
				//Owner:      wsAcc.Owner,
				//RentEpoch:  wsAcc.RentEpoch,
			},
		}
	}

}
