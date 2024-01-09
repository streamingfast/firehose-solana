package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"go.uber.org/zap"
)

type RPCFetcher struct {
	rpcClient                *rpc.Client
	latestSlot               uint64
	latestBlockRetryInterval time.Duration
	fetchInterval            time.Duration
	lastFetchAt              time.Time
	logger                   *zap.Logger
}

func NewRPC(rpcClient *rpc.Client, fetchInterval time.Duration, latestBlockRetryInterval time.Duration, logger *zap.Logger) *RPCFetcher {
	return &RPCFetcher{
		rpcClient:                rpcClient,
		fetchInterval:            fetchInterval,
		latestBlockRetryInterval: latestBlockRetryInterval,
		logger:                   logger,
	}
}

func (f *RPCFetcher) Fetch(ctx context.Context, blockNum uint64) (out *pbbstream.Block, err error) {
	f.logger.Debug("fetching block", zap.Uint64("block_num", blockNum))

	for f.latestSlot < blockNum {
		f.latestSlot, err = f.rpcClient.GetSlot(ctx, rpc.CommitmentConfirmed)
		if err != nil {
			return nil, fmt.Errorf("fetching latestSlot block num: %w", err)
		}

		f.logger.Info("got latestSlot block", zap.Uint64("latestSlot", f.latestSlot), zap.Uint64("block_num", blockNum))
		//
		if f.latestSlot < blockNum {
			time.Sleep(f.latestBlockRetryInterval)
			continue
		}
		break
	}

	blockResult, err := f.rpcClient.GetBlockWithOpts(ctx, blockNum, &rpc.GetBlockOpts{
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching block %d: %w", blockNum, err)
	}
	block := blockFromBlockResult(blockResult)
	return block, nil
}

func blockFromBlockResult(b *rpc.GetBlockResult) *pbbstream.Block {

	//todo: convert block result to pbsol.Block
	//todo: return pbbstream.Block

	//panic("implement me")
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
