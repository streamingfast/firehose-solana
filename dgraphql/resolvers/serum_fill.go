package resolvers

import (
	"encoding/hex"
	"strings"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	gtype "github.com/dfuse-io/dgraphql/types"
	"github.com/dfuse-io/solana-go"
)

type SerumFill struct {
	*pbserumhist.Fill
}

func (s SerumFill) OrderID() string { return hex.EncodeToString(s.OrderId) }
func (s SerumFill) Trader() string {
	return solana.PublicKeyFromBytes(s.Fill.Trader).String()
}
func (s SerumFill) Side() string           { return s.Fill.Side.String() }
func (s SerumFill) Market() *SerumMarket   { return nil }
func (s SerumFill) BaseToken() *Token      { return nil }
func (s SerumFill) QuoteToken() *Token     { return nil }
func (s SerumFill) LotCount() gtype.Uint64 { return 0 }
func (s SerumFill) Price() gtype.Uint64    { return 0 }
func (s SerumFill) FeeTier() string        { return strings.ToUpper(s.Fill.FeeTier.String()) }

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
