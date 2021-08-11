package resolvers

import (
	"bytes"
	"sort"

	"go.uber.org/zap"

	"github.com/dfuse-io/dfuse-solana/registry"
	"github.com/streamingfast/solana-go"
	gqlerrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/streamingfast/dgraphql"
	"github.com/streamingfast/dgraphql/types"
)

type TokensRequest struct {
	Cursor *string
	Count  *types.Uint64
}

func (r *Root) QueryTokens(request *TokensRequest) (*TokenConnection, error) {
	zlog.Debug("query tokens", zap.Reflect("request", request))
	paginator, err := dgraphql.NewPaginator(nil, nil, nil, request.Cursor, uint32(request.Count.Native()), dgraphql.IdentityCursorDecoder)
	if err != nil {
		return nil, gqlerrors.Errorf("invalid arguments: %s", err)
	}

	allTokens := r.tokensGetter()
	if len(allTokens) <= 0 {
		return &TokenConnection{Edges: nil, PageInfo: emptyPageInfo, TotalCount: 0}, nil
	}
	zlog.Debug("retrieved all tokens", zap.Int("token_count", len(allTokens)))

	sort.Slice(allTokens, func(i, j int) bool {
		metaLeft := allTokens[i].Meta
		metaRight := allTokens[j].Meta

		if metaLeft == nil && metaRight != nil {
			return false
		}

		if metaLeft != nil && metaRight == nil {
			return true
		}

		if metaLeft != nil && metaRight != nil {
			return metaLeft.Name < metaRight.Name
		}

		return bytes.Compare(allTokens[i].Address[:], allTokens[j].Address[:]) < 0
	})

	pagineableTokens := PagineableToken(allTokens)
	paginatedTokens := ([]*registry.Token)(paginator.Paginate(pagineableTokens).(PagineableToken))

	tokens := make([]*Token, len(paginatedTokens))
	for i, t := range paginatedTokens {
		tokens[i] = &Token{
			t: t,
		}
	}

	edges := make([]*TokenEdge, len(tokens))
	for i, token := range tokens {
		edges[i] = &TokenEdge{cursor: token.t.Address.String(), node: token}
	}

	hasNext := false
	if len(edges) > 0 && edges[len(edges)-1].node.t.Address.String() != allTokens[len(allTokens)-1].Address.String() {
		hasNext = true
	}

	return &TokenConnection{
		Edges:      edges,
		PageInfo:   NewPageInfoFromTokenEdges(edges, hasNext),
		TotalCount: types.Uint64(len(allTokens)),
	}, nil
}

type TokenRequest struct {
	Address string
}

func (r *Root) QueryToken(req *TokenRequest) (*TokenEdge, error) {
	pubKey, err := solana.PublicKeyFromBase58(req.Address)
	if err != nil {
		return nil, err
	}

	t := r.tokenGetter(&pubKey)
	if t == nil {
		return nil, nil
	}

	return &TokenEdge{
		cursor: t.Address.String(),
		node:   &Token{t: t},
	}, nil
}

type TokenConnection struct {
	Edges      []*TokenEdge
	PageInfo   PageInfo
	TotalCount types.Uint64
}

func NewPageInfoFromTokenEdges(edges []*TokenEdge, hasNext bool) PageInfo {
	if len(edges) == 0 {
		return emptyPageInfo
	}

	return NewPageInfo(edges[0].cursor, edges[len(edges)-1].cursor, hasNext)
}

type TokenEdge struct {
	cursor string
	node   *Token
	err    error
}

func (e *TokenEdge) Node() *Token             { return e.node }
func (e *TokenEdge) Cursor() string           { return e.cursor }
func (e *TokenEdge) SubscriptionError() error { return e.err }

type PagineableToken []*registry.Token

func (p PagineableToken) Length() int {
	return len(p)
}

func (p PagineableToken) IsEqual(index int, key string) bool {
	return p[index].Address.String() == key
}

func (p PagineableToken) Append(slice dgraphql.Pagineable, index int) dgraphql.Pagineable {
	if slice == nil {
		return dgraphql.Pagineable(PagineableToken([]*registry.Token{p[index]}))
	}

	return dgraphql.Pagineable(append(slice.(PagineableToken), p[index]))
}
