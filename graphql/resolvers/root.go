package resolvers

import (
	"github.com/dfuse-io/dfuse-solana/graphql/trade"
	"github.com/dfuse-io/solana-go/rpc"
)

// Root is the root resolvers.
type Root struct {
	rpcClient    *rpc.Client
	tradeManager *trade.Manager
	wsURL        string
}

func NewRoot(rpcClient *rpc.Client, wsURL string, manager *trade.Manager) *Root {
	return &Root{
		rpcClient:    rpcClient,
		wsURL:        wsURL,
		tradeManager: manager,
	}
}
