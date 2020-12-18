package resolvers

import (
	"context"

	pbaccounthist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/solana-go"
	gqerrs "github.com/graph-gophers/graphql-go/errors"
)

type SerumFillHistoryRequest struct {
	PubKey string
	Market *string
}

func (r *Root) QuerySerumFillHistory(ctx context.Context, request *SerumFillHistoryRequest) (out *SerumFillConnection, err error) {
	_, err = solana.PublicKeyFromBase58(request.PubKey)
	if err != nil {
		return nil, gqerrs.Errorf(`invalid "pubKey" argument %q: %w`, request.PubKey, err)
	}

	if request.Market != nil {
		_, err = solana.PublicKeyFromBase58(*request.Market)
		if err != nil {
			return nil, gqerrs.Errorf(`invalid "market" argument %q: %w`, *request.Market, err)
		}
	}

	r.serumHistoryClient.GetFills(ctx, &pbaccounthist.GetFillsRequest{})

	edges := []*SerumFillEdge{
		{cursor: "abc", node: &SerumFill{OrderID: "1", PubKey: "a", Market: SerumMarket{Address: "12", Name: "SOL/USD"}, Side: SerumSideTypeBid, BaseToken: Token{Address: "123", Name: "SOL"}, QuoteToken: Token{Address: "123", Name: "USD"}, LotCount: 10, Price: 12, FeeTier: SerumFeeTierBase}},
		{cursor: "def", node: &SerumFill{OrderID: "2", PubKey: "a", Market: SerumMarket{Address: "34", Name: "SOL/EOS"}, Side: SerumSideTypeBid, BaseToken: Token{Address: "123", Name: "SOL"}, QuoteToken: Token{Address: "456", Name: "EOS"}, LotCount: 20, Price: 15, FeeTier: SerumFeeTierSRM2}},
		{cursor: "hij", node: &SerumFill{OrderID: "3", PubKey: "a", Market: SerumMarket{Address: "ab", Name: "SOL/ETH"}, Side: SerumSideTypeAsk, BaseToken: Token{Address: "123", Name: "SOL"}, QuoteToken: Token{Address: "678", Name: "ETH"}, LotCount: 30, Price: 24, FeeTier: SerumFeeTierMSRM}},
		{cursor: "klm", node: &SerumFill{OrderID: "4", PubKey: "a", Market: SerumMarket{Address: "zf", Name: "SOL/BTC"}, Side: SerumSideTypeBid, BaseToken: Token{Address: "123", Name: "SOL"}, QuoteToken: Token{Address: "981", Name: "BTC"}, LotCount: 50, Price: 20, FeeTier: SerumFeeTierSRM4}},
		{cursor: "opq", node: &SerumFill{OrderID: "5", PubKey: "a", Market: SerumMarket{Address: "1o", Name: "SOL/DFUSE"}, Side: SerumSideTypeAsk, BaseToken: Token{Address: "123", Name: "SOL"}, QuoteToken: Token{Address: "abg", Name: "DFUSE"}, LotCount: 2, Price: 17, FeeTier: SerumFeeTierSRM6}},
	}

	return &SerumFillConnection{
		Edges:    edges,
		PageInfo: NewPageInfoFromEdges(edges),
	}, nil
}

type SerumFillEdge struct {
	cursor string
	node   *SerumFill
	err    error
}

func NewSerumFillEdge(node *SerumFill, cursor string) *SerumFillEdge {
	return &SerumFillEdge{
		cursor: cursor,
		node:   node,
	}
}

func (e *SerumFillEdge) Node() *SerumFill         { return e.node }
func (e *SerumFillEdge) Cursor() string           { return e.cursor }
func (e *SerumFillEdge) SubscriptionError() error { return e.err }

type SerumFillConnection struct {
	Edges    []*SerumFillEdge
	PageInfo PageInfo
}

func NewSerumFillConnection(edges []*SerumFillEdge, pageInfo PageInfo) *SerumFillConnection {
	return &SerumFillConnection{
		Edges:    edges,
		PageInfo: pageInfo,
	}
}

var emptyPageInfo = PageInfo{}

func NewPageInfoFromEdges(edges []*SerumFillEdge) PageInfo {
	if len(edges) == 0 {
		return emptyPageInfo
	}

	return NewPageInfo(edges[0].cursor, edges[len(edges)-1].cursor)
}
