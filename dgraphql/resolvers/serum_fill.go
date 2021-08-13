package resolvers

import (
	"fmt"
	"strings"

	"github.com/graph-gophers/graphql-go"
	gtype "github.com/streamingfast/dgraphql/types"
	pbserumhist "github.com/streamingfast/sf-solana/pb/sf/solana/serumhist/v1"
	"github.com/streamingfast/sf-solana/registry"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/serum"
	"go.uber.org/zap"
)

type SerumFill struct {
	*pbserumhist.Fill
	market     *registry.Market
	basetoken  *registry.Token
	quoteToken *registry.Token
}

func (r *Root) newSerumFill(f *pbserumhist.Fill) SerumFill {
	zlog.Debug("creating a new serum fill",
		zap.String("market_address", f.Market),
	)
	out := SerumFill{Fill: f}
	marketAddr, err := solana.PublicKeyFromBase58(f.Market)
	if err != nil {
		zlog.Warn("unable to decode public key", zap.String("address", f.Market))
		return out
	}

	market := r.marketGetter(&marketAddr)
	if market == nil {
		zlog.Warn("unknown market", zap.String("address", marketAddr.String()))
		return out
	}
	out.market = market

	baseToken := r.tokenGetter(&market.BaseToken)
	if baseToken != nil {
		out.basetoken = baseToken
	} else {
		zlog.Warn("unknown base token for market",
			zap.String("base_token", market.BaseToken.String()),
			zap.String("address", marketAddr.String()),
		)
	}

	quoteToken := r.tokenGetter(&market.QuoteToken)
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

func (s SerumFill) SlotNum() gtype.Uint64          { return gtype.Uint64(s.Fill.SlotNum) }
func (s SerumFill) TransactionIndex() gtype.Uint64 { return gtype.Uint64(s.Fill.TrxIdx) }
func (s SerumFill) InstructionIndex() gtype.Uint64 { return gtype.Uint64(s.Fill.InstIdx) }
func (s SerumFill) Timestamp() graphql.Time        { return toTime(s.Fill.Timestamp) }
func (s SerumFill) OrderNum() gtype.Uint64         { return gtype.Uint64(s.Fill.OrderSeqNum) }
func (s SerumFill) Trader() string                 { return s.Fill.Trader }

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
	orderID, err := serum.NewOrderID(s.Fill.OrderId)
	if err != nil {
		return "", fmt.Errorf("unable to get orderID: %w", err)
	}

	if s.market == nil || s.quoteToken == nil || s.basetoken == nil {
		return fmt.Sprintf("%d", orderID.Price()), nil
	}

	price := serum.PriceLotsToNumber(orderID.Price(), s.market.BaseLotSize, s.market.QuoteLotSize, uint64(s.basetoken.Decimals), uint64(s.quoteToken.Decimals))
	return price.String(), nil
}

func (s SerumFill) FeeTier() string {
	return strings.ToUpper(s.Fill.FeeTier.String())
}
