package resolvers

import "github.com/dfuse-io/dfuse-solana/registry"

type SerumMarket struct {
	Address    string
	market     *registry.Market
	basetoken  *registry.Token
	quoteToken *registry.Token
}

func (m SerumMarket) Name() *string {
	if m.market == nil {
		return nil
	}

	if m.market.Name == "" {
		return nil
	}
	return &m.market.Name
}

func (s SerumMarket) BaseToken() *Token {
	if s.basetoken != nil {
		return &Token{s.basetoken}
	}
	return nil
}

func (s SerumMarket) QuoteToken() *Token {
	if s.quoteToken != nil {
		return &Token{s.quoteToken}
	}
	return nil
}
