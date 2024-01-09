package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"go.uber.org/zap"
)

// todo: implement firecore.BlockFetcher
type RPC struct {
	rpcClient                *rpc.Client
	latest                   uint64
	latestBlockRetryInterval time.Duration
	fetchInterval            time.Duration
	lastFetchAt              time.Time
	logger                   *zap.Logger
}

func (r *RPC) Fetch(ctx context.Context, blkNum uint64) (*pbbstream.Block, error) {
	blockResult, err := r.rpcClient.GetBlock(ctx, blkNum)
	if err != nil {
		return nil, fmt.Errorf("fetching block %d: %w", blkNum, err)
	}
	block := blockFromBlockResult(blockResult)
	return block, nil
}

func blockFromBlockResult(b *rpc.GetBlockResult) *pbbstream.Block {

	panic("implement me")
	block := &pbbstream.Block{
		//Number:         b.BlockHeight,
		//Id:             "",
		//ParentId:       "",
		//Timestamp:      nil,
		//LibNum:         0,
		//PayloadKind:    0,
		//PayloadVersion: 0,
		//PayloadBuffer:  nil,
		//HeadNum:        0,
		//ParentNum:      0,
		//Payload:        nil,
	}

	return block

}
