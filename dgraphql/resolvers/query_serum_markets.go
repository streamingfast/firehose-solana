package resolvers

import (
	"bytes"
	"sort"

	"github.com/dfuse-io/dfuse-solana/registry"
	gqlerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/streamingfast/dgraphql"
	"github.com/streamingfast/dgraphql/types"
)

type SerumMarketsRequest struct {
	Cursor *string
	Count  *types.Uint64
}

func (r *Root) QuerySerumMarkets(request *SerumMarketsRequest) (*SerumMarketConnection, error) {
	paginator, err := dgraphql.NewPaginator(nil, nil, nil, request.Cursor, uint32(request.Count.Native()), dgraphql.IdentityCursorDecoder)
	if err != nil {
		return nil, gqlerrors.Errorf("invalid arguments: %s", err)
	}

	allMarkets := r.marketsGetter()
	if len(allMarkets) <= 0 {
		return &SerumMarketConnection{Edges: nil, PageInfo: emptyPageInfo, TotalCount: 0}, nil
	}

	sort.Slice(allMarkets, func(i, j int) bool {
		nameLeft := allMarkets[i].Name
		nameRight := allMarkets[j].Name

		if nameLeft == "" && nameRight != "" {
			return false
		}

		if nameLeft != "" && nameRight == "" {
			return true
		}

		if nameLeft != "" && nameRight != "" {
			return nameLeft < nameRight
		}

		return bytes.Compare(allMarkets[i].Address[:], allMarkets[j].Address[:]) < 0
	})

	pagineableMarkets := PagineableSerumMarkets(allMarkets)
	paginatedMarkets := ([]*registry.Market)(paginator.Paginate(pagineableMarkets).(PagineableSerumMarkets))

	markets := make([]*SerumMarket, len(paginatedMarkets))
	for i, t := range paginatedMarkets {
		markets[i] = &SerumMarket{
			Address:    t.Address.String(),
			market:     t,
			baseToken:  r.tokenGetter(&t.BaseToken),
			quoteToken: r.tokenGetter(&t.QuoteToken),
		}
	}

	edges := make([]*SerumMarketEdge, len(markets))
	for i, market := range markets {
		edges[i] = &SerumMarketEdge{cursor: market.Address, node: market}
	}

	hasNext := false
	if len(edges) > 0 && edges[len(edges)-1].node.market.Address.String() != allMarkets[len(allMarkets)-1].Address.String() {
		hasNext = true
	}

	return &SerumMarketConnection{
		Edges:      edges,
		PageInfo:   NewPageInfoFromSerumMarketEdges(edges, hasNext),
		TotalCount: types.Uint64(len(allMarkets)),
	}, nil
}

type SerumMarketConnection struct {
	Edges      []*SerumMarketEdge
	PageInfo   PageInfo
	TotalCount types.Uint64
}

func NewPageInfoFromSerumMarketEdges(edges []*SerumMarketEdge, hasNext bool) PageInfo {
	if len(edges) == 0 {
		return emptyPageInfo
	}

	return NewPageInfo(edges[0].cursor, edges[len(edges)-1].cursor, hasNext)
}

type SerumMarketEdge struct {
	cursor string
	node   *SerumMarket
	err    error
}

func NewSerumMarketEdge(node *SerumMarket, cursor string) *SerumMarketEdge {
	return &SerumMarketEdge{
		cursor: cursor,
		node:   node,
	}
}

func (e *SerumMarketEdge) Node() *SerumMarket       { return e.node }
func (e *SerumMarketEdge) Cursor() string           { return e.cursor }
func (e *SerumMarketEdge) SubscriptionError() error { return e.err }

type PagineableSerumMarkets []*registry.Market

func (p PagineableSerumMarkets) Length() int {
	return len(p)
}

func (p PagineableSerumMarkets) IsEqual(index int, key string) bool {
	return p[index].Address.String() == key
}

func (p PagineableSerumMarkets) Append(slice dgraphql.Pagineable, index int) dgraphql.Pagineable {
	if slice == nil {
		return dgraphql.Pagineable(PagineableSerumMarkets([]*registry.Market{p[index]}))
	}

	return dgraphql.Pagineable(append(slice.(PagineableSerumMarkets), p[index]))
}
