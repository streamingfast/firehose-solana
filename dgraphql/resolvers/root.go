package resolvers

import (
	"github.com/dfuse-io/dauth/ratelimiter"
	"github.com/dfuse-io/dfuse-solana/dgraphql/trade"
	"github.com/dfuse-io/dfuse-solana/md"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/solana-go/rpc"
)

// Root is the root resolvers of the schema
type Root struct {
	rpcClient          *rpc.Client
	tradeManager       *trade.Manager
	wsURL              string
	mdServer           *md.Server
	requestRateLimiter ratelimiter.RateLimiter
	serumHistoryClient pbserumhist.SerumHistoryClient
}

func NewRoot(
	rpcClient *rpc.Client,
	wsURL string,
	manager *trade.Manager,
	mdServer *md.Server,
	requestRateLimiter ratelimiter.RateLimiter,
	serumHistoryClient pbserumhist.SerumHistoryClient,
) (*Root, error) {
	return &Root{
		rpcClient:          rpcClient,
		wsURL:              wsURL,
		tradeManager:       manager,
		mdServer:           mdServer,
		requestRateLimiter: requestRateLimiter,
		serumHistoryClient: serumHistoryClient,
	}, nil
}
