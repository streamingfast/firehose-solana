package resolvers

import (
	"context"
	"fmt"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"

	"github.com/dfuse-io/solana-go/serum"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc/ws"
)

type TradeArgs struct {
	Account string
}


func (r *Root) Trade(ctx context.Context, args *TradeArgs) (<-chan *Trade, error) {
	zlogger := logging.Logger(ctx, zlog)

	c := make(chan *Trade)
	account := args.Account

	zlogger.Info("received trade stream", zap.String("account", account))

	//emitError := func(err error) {
	//	out := &Trade{
	//		err: dgraphql.UnwrapError(ctx, err),
	//	}
	//	c <- out
	//}


	sub := r.manager.Subscribe(account)

	go func() {
		zlog.Info("starting stream from channel")
		for {
			select {
			case <-ctx.Done():
				zlogger.Info("received context cancelled for trade")
				r.manager.Unsubscribe(sub)
				break
			case t := <- sub.Stream:
				c <- newTrade(t)
			}
		}
		close(c)
	}()


	return nil, nil
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
