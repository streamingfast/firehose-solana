package resolvers

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dfuse-solana/graphql/trade"

	"github.com/dfuse-io/logging"
	"go.uber.org/zap"

	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc/ws"
	"github.com/dfuse-io/solana-go/serum"
)

type TradeArgs struct {
	Account string
	Market  string
}

func (r *Root) Serum(ctx context.Context, args *TradeArgs) (<-chan *SerumCall, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("received trade stream", zap.String("account", args.Account))
	c := make(chan *SerumCall)
	account, err := solana.PublicKeyFromBase58(args.Account)
	if err != nil {
		return nil, fmt.Errorf("unable to parse public key: %w", err)
	}

	sub := trade.NewSubscription(account)

	go func() {
		zlog.Info("starting stream from channel")
		for {
			select {
			case <-ctx.Done():
				zlogger.Info("received context cancelled for trade")
				r.tradeManager.Unsubscribe(sub)
				return
			case t := <-sub.Stream:
				zlogger.Debug("graphql subscription received a new Instruction",
					zap.Reflect("Instruction", t),
				)

				switch i := t.Impl.(type) {
				case *serum.InstructionInitializeMarket:
					c <- &SerumCall{
						Instruction: NewSerumInitializeMarket(i),
					}
				case *serum.InstructionNewOrder:
					c <- &SerumCall{
						Instruction: NewSerumNewOrder(i),
					}
				case *serum.InstructionMatchOrder:
					c <- &SerumCall{
						Instruction: NewSerumMatchOrder(i),
					}
				case *serum.InstructionConsumeEvents:
					c <- &SerumCall{
						Instruction: NewSerumConsumeEvents(i),
					}
				case *serum.InstructionCancelOrder:
					c <- &SerumCall{
						Instruction: NewSerumCancelOrder(i),
					}
				case *serum.InstructionSettleFunds:
					c <- &SerumCall{
						Instruction: NewSerumSettleFunds(i),
					}
				case *serum.InstructionCancelOrderByClientId:
					c <- &SerumCall{
						Instruction: NewSerumCancelOrderByClientId(i),
					}
				default:
					zlog.Error(fmt.Sprintf("unknonwn insutrction type: %T", t.Impl))
				}

			}
		}
	}()

	r.tradeManager.Subscribe(sub)
	go sub.Backfill(ctx, r.rpcClient)

	return c, nil
}

type MarketRequest struct {
	MarketAddress string
}

func (r *Root) Market(ctx context.Context, args *MarketRequest) (<-chan *OrderBook, error) {
	defer func() {
		if r := recover(); r != nil {
			zlog.Error("market subscription: recovering from panic:", zap.Error(r.(error)))
		}
	}()

	zlog.Info("entering market subscription.")
	marketPublicKey := solana.MustPublicKeyFromBase58(args.MarketAddress)
	wsClient, err := ws.Dial(ctx, r.wsURL)
	if err != nil {
		return nil, fmt.Errorf("order book subscription: websocket dial: %w", err)
	}

	accountInfo, err := r.rpcClient.GetAccountInfo(ctx, marketPublicKey)
	if err != nil {
		return nil, fmt.Errorf("order book subscription: get account info: %w", err)
	}
	zlog.Debug("got account info, about to unpack", zap.Int("data_length", len(accountInfo.Value.Data)))

	var market serum.MarketV2
	err = market.Decode(accountInfo.Value.Data)
	if err != nil {
		return nil, fmt.Errorf("order book subscription: unpack market: %w", err)
	}

	c := make(chan *OrderBook)

	sub, err := wsClient.AccountSubscribe(market.Asks, "")
	if err != nil {
		return nil, fmt.Errorf("order book subscription: subscribe account info: %w", err)
	}
	go func() {
		for {
			result, err := sub.Recv()
			if err != nil {
				zlog.Error("sub.")
				//return nil, fmt.Errorf("order book subscription: subscribe account info: %w", err)
			}
			account := result.(*ws.AccountResult).Value.Account
			fmt.Println("account owner:", account.Owner)
			fmt.Println("account data:", account.Data.String())
		}
	}()

	return c, nil

}

type OrderBook struct {
	Type   SideType
	Orders []Order
}

type Order struct {
}
