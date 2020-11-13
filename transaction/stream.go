package transaction

import (
	"context"
	"fmt"

	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/rpc/ws"
	"go.uber.org/zap"
)

type TrxProcessor interface {
	Process(trx *rpc.TransactionWithMeta)
	ProcessErr(err error)
}

type Stream struct {
	rpcClient *rpc.Client
	wsURL     string
	processor TrxProcessor
}

func NewStream(rpcClient *rpc.Client, wsURL string, processor TrxProcessor) *Stream {
	return &Stream{
		rpcClient: rpcClient,
		wsURL:     wsURL,
		processor: processor,
	}
}

func (s *Stream) Launch(ctx context.Context) error {
	zlog.Info("entering Trade subscription.")
	wsClient, err := ws.Dial(ctx, s.wsURL)
	if err != nil {
		return fmt.Errorf("order book subscription: websocket dial: %w", err)
	}

	sub, err := wsClient.SlotSubscribe()
	if err != nil {
		return fmt.Errorf("order book subscription: subscribe account info: %w", err)
	}

	go func() {
		for {
			result, err := sub.Recv()
			if err != nil {
				zlog.Error("sub.")
			}
			slot := result.(*ws.SlotResult)
			//fmt.Println("slot parent:", slot.Root, slot.Parent, slot.Slot)

			block, err := s.rpcClient.GetConfirmedBlock(ctx, slot.Slot, "json")
			if err != nil {
				zlog.Error("failed to get confirmed block", zap.Uint64("slot", slot.Slot))
				s.processor.ProcessErr(err)
				continue
			}
			if block == nil {
				zlog.Error("received empty block result")
				continue
			}

			for _, trx := range block.Transactions {
				s.processor.Process(&trx)
			}
		}
	}()

	return nil
}
