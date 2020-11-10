package resolvers

import (
	"context"
	"fmt"

	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/rpc/ws"
)

type OrderBookRequest struct {
	DexAddress string
}

func (r *Root) OrderBook(ctx context.Context, args *OrderBookRequest) (<-chan *OrderBook, error) {

	dexPublicKey := solana.MustPublicKeyFromBase58(args.DexAddress)
	wsClient, err := ws.Dial(ctx, r.wsURL)
	if err != nil {
		return nil, fmt.Errorf("order book subscription: websocket dial: %w", err)
	}

	rpcClient := rpc.NewClient(r.rpcURL)

	_, err = rpcClient.GetAccountInfo(ctx, dexPublicKey)
	if err != nil {
		return nil, fmt.Errorf("order book subscription: get account info: %w", err)
	}

	sub, err := wsClient.AccountSubscribe(dexPublicKey, "")
	if err != nil {
		return nil, fmt.Errorf("order book subscription: subscribe account info: %w", err)
	}

	for {
		result, err := sub.Recv()
		if err != nil {
			return nil, fmt.Errorf("order book subscription: subscribe account info: %w", err)
		}
		account := result.(*ws.AccountResult).Value.Account
		fmt.Println("account owner:", account.Owner)
		fmt.Println("account data:", account.Data.String())
	}
}

type OrderBook struct {
	Type   string
	Orders []Order
}

type Order struct {
}
