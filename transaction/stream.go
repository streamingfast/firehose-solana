package transaction

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/rpc/ws"
	"go.uber.org/zap"
)

type TrxProcessor interface {
	Process(trx *rpc.TransactionWithMeta)
	ProcessErr(err error)
}

type Stream struct {
	rpcClient  *rpc.Client
	wsURL      string
	processor  TrxProcessor
	slotOffset uint64
}

func NewStream(rpcClient *rpc.Client, wsURL string, processor TrxProcessor, slotOffset uint64) *Stream {
	return &Stream{
		rpcClient:  rpcClient,
		wsURL:      wsURL,
		processor:  processor,
		slotOffset: slotOffset,
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

			var blockResp *rpc.GetConfirmedBlockResult
			foundBlock := false
			iter := uint64(0)
			slotID := slot.Slot - s.slotOffset
			delta := 0 * time.Second
			for foundBlock {
				time.Sleep(delta)
				iter++
				blockResp, err = s.getConfirmedBlock(ctx, slotID)
				if err != nil {
					if traceEnabled {
						zlog.Error("block cannot be confirmed... retrying in 100ms",
							zap.Uint64("slot_id", slotID),
							zap.Uint64("retry_count", iter),
						)
					}
					delta = 100 * time.Millisecond
					continue
				}
				foundBlock = true
			}
			if blockResp == nil {
				zlog.Debug("received empty block result")
				continue
			}

			for _, trx := range blockResp.Transactions {
				s.processor.Process(&trx)
			}
		}
	}()

	return nil
}

func (s *Stream) getConfirmedBlock(ctx context.Context, slotID uint64) (*rpc.GetConfirmedBlockResult, error) {
	resp, err := s.rpcClient.GetConfirmedBlock(ctx, slotID, "json")
	if err != nil {
		// block doesn't exists
		if traceEnabled {
			zlog.Error("failed to get confirmed block", zap.Uint64("slot_id", slotID))
		}
		return nil, fmt.Errorf("failed to get confirmed block at slot number %d: %w", slotID, err)
	}
	return resp, nil

}
