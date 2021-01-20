package grpc

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/solana-go"
)

func (s *Server) GetFills(ctx context.Context, request *pbserumhist.GetFillsRequest) (*pbserumhist.FillsResponse, error) {
	trader, err := solana.PublicKeyFromBase58(request.Trader)
	zlog.Debug("get fills", zap.Stringer("trader_address", trader))
	if err != nil {
		return nil, fmt.Errorf("invalid trader addresss:%s : %w", request.Trader, err)
	}

	if len(request.Market) == 0 {
		fills, err := s.manager.GetFillsByTrader(ctx, trader)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve fills: %w", err)
		}
		return &pbserumhist.FillsResponse{
			Fill: fills,
		}, nil
	}

	market, err := solana.PublicKeyFromBase58(request.Market)
	if err != nil {
		return nil, fmt.Errorf("invalid Market addresss:%s : %w", request.Trader, err)
	}
	f, err := s.manager.GetFillsByTraderAndMarket(ctx, trader, market)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve fills: %w", err)
	}
	return &pbserumhist.FillsResponse{
		Fill: f,
	}, nil
}
