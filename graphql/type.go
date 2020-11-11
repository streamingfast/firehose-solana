package graphql

import "math/big"

type SideType string

const (
	SideTypeBid string = "BID"
	SideTypeAsk string = "ASK"
)

type Trade struct {
	Market    *Market
	Side      SideType
	Size      *big.Float
	Price     *big.Float
	Liquidity *big.Float
	Fee       *big.Float
}

type Market struct {
	Name    string
	Address string
}
