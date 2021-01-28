package resolvers

import (
	"sort"
)

func (r *Root) QueryRegisteredMarkets() (out []*SerumMarket) {
	out = []*SerumMarket{}
	for _, t := range r.registryServer.GetMarkets() {
		out = append(out, &SerumMarket{
			Address:    t.Address.String(),
			market:     t,
			baseToken:  r.registryServer.GetToken(&t.QuoteToken),
			quoteToken: r.registryServer.GetToken(&t.QuoteToken),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		nameLeft := out[i].Name()
		nameRight := out[j].Name()

		if nameLeft == nil && nameRight != nil {
			return false
		}

		if nameLeft != nil && nameRight == nil {
			return true
		}

		if nameLeft != nil && nameRight != nil {
			return *nameLeft > *nameRight
		}

		return out[i].Address > out[j].Address
	})

	return
}
