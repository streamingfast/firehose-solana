package resolvers

import (
	"github.com/dfuse-io/dauth/ratelimiter"
	"github.com/dfuse-io/dfuse-solana/dgraphql/trade"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/token"
	"github.com/dfuse-io/solana-go/rpc"
)

// Root is the root resolvers of the schema
type Root struct {
	rpcClient          *rpc.Client
	tradeManager       *trade.Manager
	wsURL              string
	tokenRegistry      *token.Registry
	requestRateLimiter ratelimiter.RateLimiter
	serumHistoryClient pbserumhist.SerumHistoryClient
}

func NewRoot(
	rpcClient *rpc.Client,
	wsURL string,
	manager *trade.Manager,
	tokenRegistry *token.Registry,
	requestRateLimiter ratelimiter.RateLimiter,
	serumHistoryClient pbserumhist.SerumHistoryClient,
) (*Root, error) {
	return &Root{
		rpcClient:          rpcClient,
		wsURL:              wsURL,
		tradeManager:       manager,
		tokenRegistry:      tokenRegistry,
		requestRateLimiter: requestRateLimiter,
		serumHistoryClient: serumHistoryClient,
	}, nil
}
