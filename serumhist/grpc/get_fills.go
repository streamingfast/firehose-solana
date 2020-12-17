package grpc

import (
	"context"

	pbaccounthist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
)

func (s *Server) GetFills(ctx context.Context, request *pbaccounthist.GetFillsRequest) (*pbaccounthist.FillsResponse, error) {
	panic("implement me")
}
