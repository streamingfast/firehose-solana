package resolvers

import (
	gtype "github.com/dfuse-io/dgraphql/types"
	"github.com/dfuse-io/solana-go/serum"
)

type OrderType string

//return new EnumLayout({ limit: 0, ioc: 1, postOnly: 2 }, 4, property);
const (
	OrderTypeLimit    OrderType = "LIMIT"
	OrderTypeIOC      OrderType = "IMMEDIATE_OR_CANCEL"
	OrderTypePostOnly OrderType = "POST_ONLY"
	OrderTypeUnkown   OrderType = "UNKNOWN"
)

func NewOrderType(side uint32) OrderType {
	switch side {
	case 0:
		return OrderTypeLimit // buy
	case 1:
		return OrderTypeIOC // buy
	case 2:
		return OrderTypePostOnly // buy
	default:
		return OrderTypeUnkown
	}
}

type SideType string

//return new EnumLayout({ buy: 0, sell: 1 }, 4, property);
const (
	SideTypeBid     SideType = "BID"
	SideTypeAsk     SideType = "ASK"
	SideTypeUnknown SideType = "UNKNOWN"
)

//return new EnumLayout({ buy: 0, sell: 1 }, 4, property);

func newSideType(side uint32) SideType {
	switch side {
	case 0:
		return SideTypeBid // buy
	case 1:
		return SideTypeAsk // buy
	default:
		return SideTypeUnknown
	}
}

type SerumCall struct {
	Instruction *SerumInstruction

	err error
}

type SerumInstruction struct {
	inner interface{}
}

type SerumInitializeMarket struct {
	BaseLotSize        gtype.Uint64
	QuoteLotSize       gtype.Uint64
	FeeRateBps         gtype.Uint64
	VaultSignerNonce   gtype.Uint64
	QuoteDustThreshold gtype.Uint64
}

func (d *SerumInstruction) ToSerumInitializeMarket() (*SerumInitializeMarket, bool) {
	if v, ok := d.inner.(*SerumInitializeMarket); ok {
		return v, true
	}
	return nil, false
}

func NewSerumInitializeMarket(s *serum.InstructionInitializeMarket) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumInitializeMarket{
			BaseLotSize:        gtype.Uint64(s.BaseLotSize),
			QuoteLotSize:       gtype.Uint64(s.QuoteLotSize),
			FeeRateBps:         gtype.Uint64(s.FeeRateBps),
			VaultSignerNonce:   gtype.Uint64(s.VaultSignerNonce),
			QuoteDustThreshold: gtype.Uint64(s.QuoteLotSize),
		},
	}
}

func (d *SerumInstruction) ToSerumNewOrder() (*SerumNewOrder, bool) {
	if v, ok := d.inner.(*SerumNewOrder); ok {
		return v, true
	}
	return nil, false
}

type SerumNewOrder struct {
	Side        SideType
	LimitPrice  gtype.Uint64
	MaxQuantity gtype.Uint64
	OrderType   OrderType
	ClientID    gtype.Uint64
}

func NewSerumNewOrder(i *serum.InstructionNewOrder) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumNewOrder{
			Side:        newSideType(i.Side),
			LimitPrice:  gtype.Uint64(i.LimitPrice),
			MaxQuantity: gtype.Uint64(i.MaxQuantity),
			OrderType:   NewOrderType(i.OrderType),
			ClientID:    gtype.Uint64(i.ClientID),
		},
	}
}

type SerumMatchOrder struct {
	Limit gtype.Uint64
}

func NewSerumMatchOrder(i *serum.InstructionMatchOrder) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumMatchOrder{
			Limit: gtype.Uint64(i.Limit),
		},
	}
}

func (d *SerumInstruction) ToSerumMatchOrder() (*SerumMatchOrder, bool) {
	if v, ok := d.inner.(*SerumMatchOrder); ok {
		return v, true
	}
	return nil, false
}

type SerumConsumeEvents struct {
	Limit gtype.Uint64
}

func NewSerumConsumeEvents(i *serum.InstructionConsumeEvents) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumConsumeEvents{
			Limit: gtype.Uint64(i.Limit),
		},
	}
}

func (d *SerumInstruction) ToSerumConsumeEvents() (*SerumConsumeEvents, bool) {
	if v, ok := d.inner.(*SerumConsumeEvents); ok {
		return v, true
	}
	return nil, false
}

type SerumCancelOrder struct {
	Side          SideType
	OrderId       string
	OpenOrders    string
	OpenOrderSlot gtype.Uint64
}

func NewSerumCancelOrder(i *serum.InstructionCancelOrder) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumCancelOrder{
			Side:          newSideType(i.Side),
			OrderId:       i.OrderID.String(),
			OpenOrders:    i.OpenOrders.String(),
			OpenOrderSlot: gtype.Uint64(i.OpenOrderSlot),
		},
	}
}

func (d *SerumInstruction) ToSerumCancelOrder() (*SerumCancelOrder, bool) {
	if v, ok := d.inner.(*SerumCancelOrder); ok {
		return v, true
	}
	return nil, false
}

type SerumSettleFunds struct {
}

func NewSerumSettleFunds(i *serum.InstructionSettleFunds) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumSettleFunds{},
	}
}

func (d *SerumInstruction) ToSerumSettleFunds() (*SerumSettleFunds, bool) {
	if v, ok := d.inner.(*SerumSettleFunds); ok {
		return v, true
	}
	return nil, false
}

type SerumCancelOrderByClientId struct {
	ClientID gtype.Uint64
}

func NewSerumCancelOrderByClientId(i *serum.InstructionCancelOrderByClientId) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumCancelOrderByClientId{
			ClientID: gtype.Uint64(i.ClientID),
		},
	}
}

func (d *SerumInstruction) ToSerumCancelOrderByClientId() (*SerumCancelOrderByClientId, bool) {
	if v, ok := d.inner.(*SerumCancelOrderByClientId); ok {
		return v, true
	}
	return nil, false
}
