package grpc

import (
	"context"
	"fmt"

	pbserum "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serum/v1"
	pbaccounthist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/solana-go"
	"github.com/golang/protobuf/proto"
)

func (s *Server) GetFills(ctx context.Context, request *pbaccounthist.GetFillsRequest) (*pbaccounthist.FillsResponse, error) {

	market, err := solana.PublicKeyFromBase58(request.Market)
	if err != nil {
		return nil, fmt.Errorf("invalid market address: %w", err)
	}

	trader, err := solana.PublicKeyFromBase58(request.Trader)
	if err != nil {
		return nil, fmt.Errorf("invalid trader address: %w", err)
	}

	orderKeyPrefixes := keyer.EncodeOrdersPrefixByMarketPubkey(trader, market)

	orderIterator := s.kvStore.Prefix(ctx, orderKeyPrefixes, 100)

	var fillKeys [][]byte
	for orderIterator.Next() {
		k := orderIterator.Item().Key
		_, market, orderSeqNum, slotNum := keyer.DecodeOrdersByMarketPubkey(k)
		fk := keyer.EncodeFillData(market, orderSeqNum, slotNum)
		fillKeys = append(fillKeys, fk)
	}

	if orderIterator.Err() != nil {
		return nil, fmt.Errorf("failed to iterate orders: %w", err)
	}

	fillsIterator := s.kvStore.BatchGet(ctx, fillKeys)

	var fills []*pbserum.Fill
	for fillsIterator.Next() {
		v := orderIterator.Item().Value
		f := &pbserum.Fill{}
		err := proto.Unmarshal(v, f)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal order: %w", err)
		}
		fills = append(fills, f)
	}

	if orderIterator.Err() != nil {
		return nil, fmt.Errorf("failed to iterate fills: %w", err)
	}

	return &pbaccounthist.FillsResponse{
		Fill: fills,
	}, nil
}
