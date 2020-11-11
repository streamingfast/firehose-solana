package resolvers

import "github.com/dfuse-io/solana-go/rpc"

import "github.com/dfuse-io/dfuse-solana/graphql"

// Root is the root resolvers.
type Root struct {
	rpcClient *rpc.Client
	manager   *graphql.Manager
	wsURL     string
}

func NewRoot(rpcClient *rpc.Client, wsURL string, manager *graphql.Manager) *Root {
	return &Root{
		rpcClient: rpcClient,
		wsURL:     wsURL,
		manager:   manager,
	}
}
