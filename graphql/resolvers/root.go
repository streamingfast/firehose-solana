package resolvers

import "github.com/dfuse-io/solana-go/rpc"

// Root is the root resolvers.
type Root struct {
	rpcClient *rpc.Client
	wsURL     string
}

func NewRoot(rpcClient *rpc.Client, wsURL string) *Root {
	return &Root{
		rpcClient: rpcClient,
		wsURL:     wsURL,
	}
}
