package resolvers

import (
	"fmt"
	"strings"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/registry"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/serum"
	"github.com/graph-gophers/graphql-go"
	"go.uber.org/zap"
)

type SerumFill struct {
	*pbserumhist.Fill
	market     *registry.Market
	basetoken  *registry.Token
	quoteToken *registry.Token
}

func (r *Root) newSerumFill(f *pbserumhist.Fill, reg *registry.Server) SerumFill {
	zlog.Debug("creating a new serum fill",
		zap.String("market_address", f.Market),
	)
	out := SerumFill{Fill: f}
	marketAddr, err := solana.PublicKeyFromBase58(f.Market)
	if err != nil {
		zlog.Warn("unable to decode public key", zap.String("address", f.Market))
		return out
	}

	market := reg.GetMarket(&marketAddr)
	if market == nil {
		zlog.Warn("unknown market", zap.String("address", marketAddr.String()))
		return out
	}
	out.market = market

	baseToken := reg.GetToken(&market.BaseToken)
	if baseToken != nil {
		out.basetoken = baseToken
	} else {
		zlog.Warn("unknown base token for market",
			zap.String("base_token", market.BaseToken.String()),
			zap.String("address", marketAddr.String()),
		)
	}

	quoteToken := reg.GetToken(&market.QuoteToken)
	if quoteToken != nil {
		out.quoteToken = quoteToken
	} else {
		zlog.Warn("unknown quote token for market",
			zap.String("quote_token", market.QuoteToken.String()),
			zap.String("address", marketAddr.String()),
		)
	}

	return out
}

func (s SerumFill) Timestamp() graphql.Time { return toTime(s.Fill.Timestamp) }
func (s SerumFill) OrderID() string         { return s.OrderId }
func (s SerumFill) Trader() string          { return s.Fill.Trader }

// func (s SerumFill) SeqNum() (commonTypes.Uint64, error) {
// 	seqNo, err := serum.GetSeqNum(s.OrderId, s.Fill.Side)
// 	return commonTypes.Uint64(seqNo), err
// }

func (s SerumFill) Market() SerumMarket {
	return SerumMarket{
		Address:    s.Fill.Market,
		market:     s.market,
		baseToken:  s.basetoken,
		quoteToken: s.quoteToken,
	}
}

func (s SerumFill) Side() string { return s.Fill.Side.String() }

func (s SerumFill) QuantityPaid() TokenAmount {
	token := s.basetoken
	if s.Fill.Side == pbserumhist.Side_BID {
		token = s.quoteToken
	}
	return TokenAmount{
		t: token,
		v: s.NativeQtyPaid,
	}
}

func (s SerumFill) QuantityReceived() TokenAmount {
	token := s.quoteToken
	if s.Fill.Side == pbserumhist.Side_BID {
		token = s.basetoken
	}
	return TokenAmount{
		t: token,
		v: s.NativeQtyReceived,
	}
}

func (s SerumFill) Price() (string, error) {
	p, err := serum.GetPrice(s.Fill.OrderId)
	if err != nil {
		return "", err
	}
	if s.market == nil || s.quoteToken == nil || s.basetoken == nil {
		return fmt.Sprintf("%d", p), nil
	}

	price := serum.PriceLotsToNumber(p, s.market.BaseLotSize, s.market.QuoteLotSize, uint64(s.basetoken.Decimals), uint64(s.quoteToken.Decimals))
	return price.String(), nil
}

func (s SerumFill) FeeTier() string {
	return strings.ToUpper(s.Fill.FeeTier.String())
}
