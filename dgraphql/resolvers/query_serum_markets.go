package resolvers

import (
	"sort"
)

type SerumMarketsRequest struct {
	Cursor *string
}

func (r *Root) QuerySerumMarkets(request *SerumMarketsRequest) (*SerumMarketConnection, error) {
	// FIXME: Improve by limiting to specific value
	markets := []*SerumMarket{}
	for _, t := range r.registryServer.GetMarkets() {
		markets = append(markets, &SerumMarket{
			Address:    t.Address.String(),
			market:     t,
			baseToken:  r.registryServer.GetToken(&t.QuoteToken),
			quoteToken: r.registryServer.GetToken(&t.QuoteToken),
		})
	}

	sort.Slice(markets, func(i, j int) bool {
		nameLeft := markets[i].Name()
		nameRight := markets[j].Name()

		if nameLeft == nil && nameRight != nil {
			return false
		}

		if nameLeft != nil && nameRight == nil {
			return true
		}

		if nameLeft != nil && nameRight != nil {
			return *nameLeft > *nameRight
		}

		return markets[i].Address > markets[j].Address
	})

	edges := make([]*SerumMarketEdge, len(markets))
	for i, market := range markets {
		edges[i] = &SerumMarketEdge{cursor: market.Address, node: market}
	}

	return &SerumMarketConnection{
		Edges:    edges,
		PageInfo: NewPageInfoFromSerumMarketEdges(edges),
	}, nil
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

type SerumMarketConnection struct {
	Edges    []*SerumMarketEdge
	PageInfo PageInfo
}

func NewSerumMarketConnection(edges []*SerumMarketEdge, pageInfo PageInfo) *SerumMarketConnection {
	return &SerumMarketConnection{
		Edges:    edges,
		PageInfo: pageInfo,
	}
}

func NewPageInfoFromSerumMarketEdges(edges []*SerumMarketEdge) PageInfo {
	if len(edges) == 0 {
		return emptyPageInfo
	}

	return NewPageInfo(edges[0].cursor, edges[len(edges)-1].cursor)
}
