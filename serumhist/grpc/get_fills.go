package grpc

import (
	"context"
	"fmt"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/solana-go"
	"github.com/golang/protobuf/proto"
)

func (s *Server) GetFills(ctx context.Context, request *pbserumhist.GetFillsRequest) (*pbserumhist.FillsResponse, error) {
	market := solana.PublicKeyFromBytes(request.Market)
	trader := solana.PublicKeyFromBytes(request.Trader)

	orderIterator := s.kvStore.Prefix(ctx, keyer.EncodeOrdersPrefixByMarketPubkey(trader, market), 100)

	var fillKeys [][]byte
	for orderIterator.Next() {
		k := orderIterator.Item().Key
		_, market, orderSeqNum, slotNum := keyer.DecodeOrdersByMarketPubkey(k)
		fk := keyer.EncodeFillData(market, orderSeqNum, slotNum)
		fillKeys = append(fillKeys, fk)
	}

	if err := orderIterator.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate orders: %w", err)
	}

	getFillsCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	fillsIterator := s.kvStore.BatchGet(getFillsCtx, fillKeys)

	var fills []*pbserumhist.Fill
	for fillsIterator.Next() {
		f := &pbserumhist.Fill{}
		err := proto.Unmarshal(orderIterator.Item().Value, f)
		if err != nil {
			fillsIterator.PushFinished()
			return nil, fmt.Errorf("failed to unmarshal order: %w", err)
		}

		fills = append(fills, f)
	}

	if err := orderIterator.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate fills: %w", err)
	}

	return &pbserumhist.FillsResponse{
		Fill: fills,
	}, nil
}
