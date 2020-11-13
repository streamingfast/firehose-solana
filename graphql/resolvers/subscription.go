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
}

func (r *Root) Trade(ctx context.Context, args *TradeArgs) (<-chan *Trade, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("received trade stream", zap.String("account", args.Account))
	c := make(chan *Trade)
	account, err := solana.PublicKeyFromBase58(args.Account)
	if err != nil {
		return nil, fmt.Errorf("unable to parse public key: %w", err)
	}

	sub := trade.NewSubscription(account)

	go func() {
		zlog.Info("starting stream from channel")
		for {
			zlog.Info("waiting for instruction")
			select {
			case <-ctx.Done():
				zlogger.Info("received context cancelled for trade")
				r.tradeManager.Unsubscribe(sub)
				break
			case t := <-sub.Stream:
				zlogger.Info("graphql subscription received a new instruction")
				c <- newTrade(t)
				zlogger.Info("sent graphql new instruction subscription")
			}
		}
		zlog.Info("clsoing channel")
		close(c)
	}()

	r.tradeManager.Subscribe(sub)
	sub.Backfill(ctx, r.rpcClient)

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
