package resolvers

import (
	"github.com/dfuse-io/dfuse-solana/token"
	"github.com/dfuse-io/dgraphql/types"
	"github.com/dfuse-io/solana-go"
)

func (r *Root) QueryTokens() (out []*TokensResponse) {
	out = []*TokensResponse{}
	for _, t := range r.tokenRegistry.GetTokens() {
		out = append(out, TokensResponseFromRegistryEntry(t))
	}

	return
}

func (r *Root) QueryToken(req *TokenRequest) (*TokensResponse, error) {
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

type TokensResponse struct {
	Address         string
	MintAddress     string
	MintAuthority   string
	FreezeAuthority string
	supply          uint64
	Decimals        int32
}

func (t *TokensResponse) Supply() types.Uint64 { return types.Uint64(t.supply) }
