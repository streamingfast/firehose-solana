package resolvers

import (
	"github.com/dfuse-io/dfuse-eosio/dgraphql/types"
	"github.com/dfuse-io/dfuse-solana/token"
	"github.com/dfuse-io/solana-go"
)

type TokenRequest struct {
	Address string
}

func (r *Root) RegisteredToken(req *TokenRequest) (*RegisteredTokensResponse, error) {
	pubKey, err := solana.PublicKeyFromBase58(req.Address)
	if err != nil {
		return nil, err
	}
	t := r.tokenRegistry.GetToken(&pubKey)
	if t == nil || t.Meta == nil {
		return nil, nil
	}
	return RegisteredTokensResponseFromRegistryEntry(t), nil
}

func (r *Root) RegisteredTokens() (out []*RegisteredTokensResponse) {
	out = []*RegisteredTokensResponse{}
	for _, t := range r.tokenRegistry.GetTokens() {
		if t.Meta != nil {
			out = append(out, RegisteredTokensResponseFromRegistryEntry(t))
		}
	}
	return
}
func (r *Root) Token(req *TokenRequest) (*TokensResponse, error) {
	pubKey, err := solana.PublicKeyFromBase58(req.Address)
	if err != nil {
		return nil, err
	}
	t := r.tokenRegistry.GetToken(&pubKey)
	if t == nil {
		return nil, nil
	}
	return TokensResponseFromRegistryEntry(t), nil
}

func (r *Root) Tokens() (out []*TokensResponse) {
	out = []*TokensResponse{}
	for _, t := range r.tokenRegistry.GetTokens() {
		out = append(out, TokensResponseFromRegistryEntry(t))
	}
	return
}

type TokensResponse struct {
	Address         string
	MintAddress     string
	MintAuthority   string
	FreezeAuthority string
	supply          uint64
	Decimals        int32
}

func (t *TokensResponse) Supply() types.Uint64 { return types.Uint64(t.supply) }

type RegisteredTokensResponse struct {
	Address         string
	MintAddress     string
	MintAuthority   string
	FreezeAuthority string
	supply          uint64
	Decimals        int32
	Symbol          string
	Name            string
	Logo            string
}

func (t *RegisteredTokensResponse) Supply() types.Uint64 { return types.Uint64(t.supply) }

func RegisteredTokensResponseFromRegistryEntry(token *token.RegisteredToken) *RegisteredTokensResponse {

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
	}
	return r
}
func TokensResponseFromRegistryEntry(token *token.RegisteredToken) *TokensResponse {

	r := &TokensResponse{
		Address:         token.Address.String(),
		MintAuthority:   token.MintAuthority.String(),
		FreezeAuthority: token.FreezeAuthority.String(),
		supply:          uint64(token.Supply),
		Decimals:        int32(token.Decimals),
	}
	return r
}
