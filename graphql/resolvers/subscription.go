package resolvers

import (
	"bytes"
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/lunixbochs/struc"

	"github.com/dfuse-io/solana-go/serum"

	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/rpc/ws"
)

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

	_ = wsClient

	rpcClient := rpc.NewClient(r.rpcURL)

	accountInfo, err := rpcClient.GetAccountInfo(ctx, marketPublicKey)
	if err != nil {
		return nil, fmt.Errorf("order book subscription: get account info: %w", err)
	}
	zlog.Debug("got account info, about to unpack", zap.Int("data_length", len(accountInfo.Value.Data)))

	var market serum.MarketV2
	err = struc.Unpack(bytes.NewReader(accountInfo.Value.Data), &market)
	if err != nil {
		return nil, fmt.Errorf("order book subscription: unpack market: %w", err)
	}
	zlog.Debug("market unpacked")
	fmt.Println("market:", market)

	c := make(chan *OrderBook)

	sub, err := wsClient.AccountSubscribe(marketPublicKey, "")
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
	Type   string
	Orders []Order
}

type Order struct {
}
