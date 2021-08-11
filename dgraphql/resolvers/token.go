package resolvers

import (
	"github.com/dfuse-io/dfuse-solana/registry"
	"github.com/streamingfast/solana-go"
	gtype "github.com/streamingfast/dgraphql/types"
)

type Token struct {
	address *solana.PublicKey
	t       *registry.Token
}

func (t Token) Address() string {
	if t.t == nil {
		return t.address.String()
	}

	return t.t.Address.String()
}

func (t Token) MintAuthority() *string {
	if t.t == nil {
		return nil
	}

	v := t.t.MintAuthority.String()
	return &v
}

func (t Token) FreezeAuthority() *string {
	if t.t == nil {
		return nil
	}

	v := t.t.FreezeAuthority.String()
	return &v
}

func (t Token) Supply() *gtype.Uint64 {
	if t.t == nil {
		return nil
	}

	v := gtype.Uint64(t.t.Supply)
	return &v
}

func (t Token) Decimals() *int32 {
	if t.t == nil {
		return nil
	}

	v := int32(t.t.Decimals)
	return &v
}

func (t Token) Verified() bool {
	if t.t == nil {
		return false
	}

	return t.t.Verified
}

func (t Token) Meta() *TokenMeta {
	if t.t != nil && t.t.Meta != nil {
		return &TokenMeta{
			Symbol:  t.t.Meta.Symbol,
			Name:    t.t.Meta.Name,
			logo:    t.t.Meta.Logo,
			website: t.t.Meta.Website,
		}
	}
	return nil
}

type TokenMeta struct {
	Symbol  string
	Name    string
	logo    string
	website string
}

func (t TokenMeta) Logo() *string {
	if t.logo == "" {
		return nil
	}
	return &t.logo
}

func (t TokenMeta) Website() *string {
	if t.website == "" {
		return nil
	}
	return &t.website
}

type TokenAmount struct {
	t *registry.Token
	v uint64
}

func (t TokenAmount) Token() *Token {
	if t.t != nil {
		return &Token{nil, t.t}
	}
	return nil
}

func (t TokenAmount) Value() gtype.Uint64 {
	return gtype.Uint64(t.v)

}

func (t TokenAmount) Display() *string {
	if t.t == nil {
		return nil
	}

	s := t.t.Display(t.v)
	return &s
}
