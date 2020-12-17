package resolvers

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-solana/dgraphql/trade"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/serum"
	"go.uber.org/zap"
)

type TradeArgs struct {
	Account string
}

func (r *Root) SubscriptionSerumInstructionHistory(ctx context.Context, args *TradeArgs) (<-chan *SerumInstructionResponse, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("received trade stream", zap.String("account", args.Account))

	account, err := solana.PublicKeyFromBase58(args.Account)
	if err != nil {
		return nil, fmt.Errorf("unable to parse public key: %w", err)
	}

	c := make(chan *SerumInstructionResponse)
	sub := trade.NewSubscription(account)

	go func() {
		zlog.Info("starting stream from channel")
		for {
			select {
			case <-ctx.Done():
				zlogger.Info("received context cancelled for trade")
				r.tradeManager.Unsubscribe(sub)
				return
			case t, ok := <-sub.Stream:
				if !ok {
					if sub.Err != nil {
						c <- &SerumInstructionResponse{err: sub.Err}
						return
					}

					close(c)
					return
				}

				zlogger.Debug("graphql subscription received a new instruction")
				s := &SerumInstructionResponse{
					TrxSignature: t.TrxSignature,
					trxError:     t.TrxError,
				}

				if t.Decoded == nil {
					s.Instruction = NewUndecodedInstruction(t.Compiled, "unable to decode serum instruction")
					c <- s
					continue
				}

				switch i := t.Decoded.Impl.(type) {
				case *serum.InstructionInitializeMarket:
					s.Instruction = NewSerumInitializeMarket(i)
					c <- s
				case *serum.InstructionNewOrder:
					s.Instruction = NewSerumNewOrder(i)
					c <- s
				case *serum.InstructionMatchOrder:
					s.Instruction = NewSerumMatchOrder(i)
					c <- s
				case *serum.InstructionConsumeEvents:
					s.Instruction = NewSerumConsumeEvents(i)
					c <- s
				case *serum.InstructionCancelOrder:
					s.Instruction = NewSerumCancelOrder(i)
					c <- s
				case *serum.InstructionSettleFunds:
					s.Instruction = NewSerumSettleFunds(i)
					c <- s
				case *serum.InstructionCancelOrderByClientId:
					s.Instruction = NewSerumCancelOrderByClientId(i)
					c <- s
				default:
					zlog.Error(fmt.Sprintf("unknown instruction %T", t.Decoded.Impl))
				}
			}
		}
	}()

	r.tradeManager.Subscribe(sub)
	go sub.Backfill(ctx, r.rpcClient)

	return c, nil
}
