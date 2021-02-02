package resolvers

import (
	"github.com/dfuse-io/dfuse-solana/registry"
	gtype "github.com/dfuse-io/dgraphql/types"
)

type Token struct {
	t *registry.Token
}

func (t Token) Address() string { return t.t.Address.String() }

func (t Token) MintAuthority() string {
	return t.t.MintAuthority.String()
}

func (t Token) FreezeAuthority() string {
	return t.t.FreezeAuthority.String()
}

func (t Token) Supply() gtype.Uint64 {
	return gtype.Uint64(t.t.Supply)
}

func (t Token) Decimals() int32 {
	return int32(t.t.Decimals)
}

func (t Token) Verified() bool {
	return t.t.Verified
}

func (t Token) Meta() *TokenMeta {
	if t.t.Meta != nil {
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
		return &Token{t.t}
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
