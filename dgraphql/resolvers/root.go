package resolvers

import (
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/registry"
	serumztics "github.com/dfuse-io/dfuse-solana/serumviz/analytics"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/streamingfast/dauth/ratelimiter"
)

// Root is the root resolvers of the schema
type Root struct {
	rpcClient           *rpc.Client
	wsURL               string
	requestRateLimiter  ratelimiter.RateLimiter
	serumHistoryClient  pbserumhist.SerumHistoryClient
	serumhistAnalyzable serumztics.Analyzable

	// Interfaces we use internally for testing purposes
	marketGetter  func(address *solana.PublicKey) *registry.Market
	marketsGetter func() []*registry.Market
	tokenGetter   func(in *solana.PublicKey) *registry.Token
	tokensGetter  func() []*registry.Token
}

func NewRoot(
	rpcClient *rpc.Client,
	wsURL string,
	mdServer *registry.Server,
	serumhistAnalytic serumztics.Analyzable,
	requestRateLimiter ratelimiter.RateLimiter,
	serumHistoryClient pbserumhist.SerumHistoryClient,

) (*Root, error) {
	return &Root{
		rpcClient:           rpcClient,
		wsURL:               wsURL,
		requestRateLimiter:  requestRateLimiter,
		serumHistoryClient:  serumHistoryClient,
		serumhistAnalyzable: serumhistAnalytic,

		marketGetter:  mdServer.GetMarket,
		marketsGetter: mdServer.GetMarkets,
		tokenGetter:   mdServer.GetToken,
		tokensGetter:  mdServer.GetTokens,
	}, nil
}
