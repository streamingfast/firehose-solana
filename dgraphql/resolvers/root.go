package resolvers

import (
	"github.com/streamingfast/dauth/ratelimiter"
	"github.com/streamingfast/solana-go/rpc"
)

// Root is the root resolvers of the schema
type Root struct {
	rpcClient          *rpc.Client
	wsURL              string
	requestRateLimiter ratelimiter.RateLimiter
}

func NewRoot(
	rpcClient *rpc.Client,
	wsURL string,
	requestRateLimiter ratelimiter.RateLimiter,

) (*Root, error) {
	return &Root{
		rpcClient:          rpcClient,
		wsURL:              wsURL,
		requestRateLimiter: requestRateLimiter,
	}, nil
}
