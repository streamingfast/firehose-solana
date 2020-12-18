package resolvers

import gtype "github.com/dfuse-io/dgraphql/types"

type SerumFill struct {
	OrderID    string
	PubKey     string
	Market     SerumMarket
	Side       SerumSideType
	BaseToken  Token
	QuoteToken Token
	LotCount   gtype.Uint64
	Price      gtype.Uint64
	FeeTier    SerumFeeTier
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
