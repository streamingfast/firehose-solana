package resolvers

import (
	"github.com/dfuse-io/dfuse-solana/md"
	"github.com/dfuse-io/dgraphql/types"
	"github.com/dfuse-io/solana-go"
)

func (r *Root) QueryRegisteredTokens() (out []*RegisteredTokensResponse) {
	out = []*RegisteredTokensResponse{}
	for _, t := range r.mdServer.GetTokens() {
		if t.Meta != nil {
			out = append(out, RegisteredTokensResponseFromRegistryEntry(t))
		}
	}
	return
}

func (r *Root) QueryRegisteredToken(req *TokenRequest) (*RegisteredTokensResponse, error) {
	pubKey, err := solana.PublicKeyFromBase58(req.Address)
	if err != nil {
		return nil, err
	}
	t := r.mdServer.GetToken(&pubKey)
	if t == nil || t.Meta == nil {
		return nil, nil
	}
	return RegisteredTokensResponseFromRegistryEntry(t), nil
}

func RegisteredTokensResponseFromRegistryEntry(token *md.RegisteredToken) *RegisteredTokensResponse {
	r := &RegisteredTokensResponse{
		Address:         token.Address.String(),
		MintAuthority:   token.MintAuthority.String(),
		FreezeAuthority: token.FreezeAuthority.String(),
		supply:          uint64(token.Supply),
		Decimals:        int32(token.Decimals),
	}
	if token.Meta != nil {
		r.Symbol = token.Meta.Symbol
		r.Name = token.Meta.Name
		r.Logo = token.Meta.Logo
		r.Website = token.Meta.Website
	}
	return r
}

type RegisteredTokensResponse struct {
	Address         string
	MintAuthority   string
	FreezeAuthority string
	supply          uint64
	Decimals        int32
	Symbol          string
	Name            string
	Logo            string
	Website         string
}

func (t *RegisteredTokensResponse) Supply() types.Uint64 { return types.Uint64(t.supply) }
