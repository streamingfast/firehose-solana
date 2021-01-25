package resolvers

import (
	"strings"

	"github.com/dfuse-io/solana-go"
	"go.uber.org/zap"

	"github.com/dfuse-io/dfuse-solana/registry"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	gtype "github.com/dfuse-io/dgraphql/types"
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

	m := reg.GetMarket(&marketAddr)
	if m == nil {
		zlog.Warn("unknown market", zap.String("address", marketAddr.String()))
		out.market = &registry.Market{Address: marketAddr}
		return out
	}

	out.market = m

	baseToken := reg.GetToken(&m.BaseToken)
	if baseToken == nil {
		zlog.Warn("unknown base token for market",
			zap.String("base_token", m.BaseToken.String()),
			zap.String("address", marketAddr.String()),
		)
		out.basetoken = &registry.Token{Address: m.BaseToken}
	} else {
		out.basetoken = baseToken
	}

	quoteToken := reg.GetToken(&m.QuoteToken)
	if quoteToken == nil {
		zlog.Warn("unknown quote token for market",
			zap.String("quote_token", m.QuoteToken.String()),
			zap.String("address", marketAddr.String()),
		)
		out.quoteToken = &registry.Token{Address: m.QuoteToken}
	} else {
		out.quoteToken = quoteToken
	}

	return out
}

func (s SerumFill) OrderID() string {
	return s.OrderId
}

func (s SerumFill) Trader() string {
	return s.Fill.Trader
}

func (s SerumFill) Side() string {
	return s.Fill.Side.String()
}

func (s SerumFill) Market() *SerumMarket {
	return &SerumMarket{
		Address: s.market.Address.String(),
		Name:    s.market.Name,
	}

}
func (s SerumFill) BaseToken() *Token {
	t := &Token{Address: s.basetoken.Address.String()}
	if s.basetoken.Meta != nil {
		t.Name = s.basetoken.Meta.Name
	}
	return t
}

func (s SerumFill) QuoteToken() *Token {
	t := &Token{Address: s.quoteToken.Address.String()}
	if s.quoteToken.Meta != nil {
		t.Name = s.quoteToken.Meta.Name
	}
	return t
}
func (s SerumFill) LotCount() gtype.Uint64 {
	return 0
}
func (s SerumFill) Price() (gtype.Uint64, error) {
	v, err := s.Fill.GetPrice()
	if err != nil {
		return 0, err
	}
	return gtype.Uint64(v), nil

}
func (s SerumFill) FeeTier() string {
	return strings.ToUpper(s.Fill.FeeTier.String())
}

type SerumFeeTier = string

const (
	SerumFeeTierBase SerumFeeTier = "BASE"
	SerumFeeTierSRM2              = "SRM2"
	SerumFeeTierSRM3              = "SRM3"
	SerumFeeTierSRM4              = "SRM4"
	SerumFeeTierSRM5              = "SRM5"
	SerumFeeTierSRM6              = "SRM6"
	SerumFeeTierMSRM              = "MSRM"
)
