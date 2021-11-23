package resolvers

import (
	"context"
	"fmt"

	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/serum"
	"github.com/streamingfast/solana-go/rpc/ws"
	"go.uber.org/zap"
)

type SerumMarketSubscription struct {
	MarketAddress string
}

func (r *Root) SubscriptionMarket(ctx context.Context, args *SerumMarketSubscription) (<-chan *SerumOrderBook, error) {
	zlog.Info("entering market subscription.")
	marketPublicKey := solana.MustPublicKeyFromBase58(args.MarketAddress)
	wsClient := ws.NewClient(r.wsURL, false)

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

	c := make(chan *SerumOrderBook)

	sub, err := wsClient.AccountSubscribe(market.Asks, "")
	if err != nil {
		return nil, fmt.Errorf("order book subscription: subscribe account info: %w", err)
	}
	go func() {
		for {
			result, err := sub.Recv(ctx)
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

type SerumOrderBook struct {
	Type   SerumSideType
	Orders []SerumOrder
}

type SerumOrder struct {
}

type SerumSideType string

const (
	SerumSideTypeBid     SerumSideType = "BID"
	SerumSideTypeAsk                   = "ASK"
	SerumSideTypeUnknown               = "UNKNOWN"
)
