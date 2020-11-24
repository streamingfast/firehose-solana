package resolvers

import (
	"github.com/dfuse-io/dfuse-eosio/dgraphql/types"
	"github.com/dfuse-io/dfuse-solana/token"
	"github.com/dfuse-io/solana-go"
)

func (r *Root) SplTokens() (out []*SplTokensResponse) {
	out = []*SplTokensResponse{}
	for _, t := range r.tokenRegistry.GetTokens() {
		out = append(out, SplTokensResponseFromRegistryEntry(t))
	}
	return
}

type SplTokenRequest struct {
	Address string
}

func (r *Root) SplToken(req *SplTokenRequest) (*SplTokensResponse, error) {
	pubKey, err := solana.PublicKeyFromBase58(req.Address)
	if err != nil {
		return nil, err
	}
	t := r.tokenRegistry.GetToken(&pubKey)
	return SplTokensResponseFromRegistryEntry(t), nil
}

type SplTokensResponse struct {
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

func (t *SplTokensResponse) Supply() types.Uint64 { return types.Uint64(t.supply) }

func SplTokensResponseFromRegistryEntry(token *token.RegisteredToken) *SplTokensResponse {

	r := &SplTokensResponse{
		Address:         token.Address.String(),
		MintAddress:     token.Mint.MintAuthority.String(),
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
