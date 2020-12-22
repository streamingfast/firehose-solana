package grpc

import (
	"context"
	"fmt"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/solana-go"
)

func (s *Server) GetFills(ctx context.Context, request *pbserumhist.GetFillsRequest) (*pbserumhist.FillsResponse, error) {
	trader := solana.PublicKeyFromBytes(request.Trader)

	if len(request.Market) == 0 {
		fills, err := s.manager.GetFillsByTrader(ctx, trader)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve fills: %w", err)
		}
		return &pbserumhist.FillsResponse{
			Fill: fills,
		}, nil
	}

	market := solana.PublicKeyFromBytes(request.Market)
	f, err := s.manager.GetFillsByTraderAndMarket(ctx, trader, market)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve fills: %w", err)
	}
	return &pbserumhist.FillsResponse{
		Fill: f,
	}, nil
}
