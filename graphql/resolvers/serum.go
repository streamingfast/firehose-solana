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

type SerumResponse struct {
	Instruction *SerumInstruction
	err         error
}

type SerumInstruction struct {
	inner interface{}
}

type SerumInitializeMarketAccounts struct {
	Market        string
	SplCoinToken  string
	SplPriceToken string
	CoinMint      string
	PriceMint     string
}
type SerumInitializeMarket struct {
	BaseLotSize        gtype.Uint64
	QuoteLotSize       gtype.Uint64
	FeeRateBps         gtype.Uint64
	VaultSignerNonce   gtype.Uint64
	QuoteDustThreshold gtype.Uint64

	Accounts SerumInitializeMarketAccounts
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
			Accounts: SerumInitializeMarketAccounts{
				Market:        s.Accounts.Market.PublicKey.String(),
				SplCoinToken:  s.Accounts.SPLCoinToken.PublicKey.String(),
				SplPriceToken: s.Accounts.SPLPriceToken.PublicKey.String(),
				CoinMint:      s.Accounts.CoinMint.PublicKey.String(),
				PriceMint:     s.Accounts.PriceMint.PublicKey.String(),
			},
		},
	}
}

func (d *SerumInstruction) ToSerumNewOrder() (*SerumNewOrder, bool) {
	if v, ok := d.inner.(*SerumNewOrder); ok {
		return v, true
	}
	return nil, false
}

type SerumNewOrderAccounts struct {
	Market          string
	OpenOrders      string
	RequestQueue    string
	Payer           string
	Owner           string
	CoinVault       string
	PcVault         string
	SplTokenProgram string
	Rent            string
	SRMDiscount     *string
}
type SerumNewOrder struct {
	Side        SideType
	LimitPrice  gtype.Uint64
	MaxQuantity gtype.Uint64
	OrderType   OrderType
	ClientID    gtype.Uint64

	Accounts SerumNewOrderAccounts
}

func NewSerumNewOrder(i *serum.InstructionNewOrder) *SerumInstruction {
	s := &SerumNewOrder{
		Side:        newSideType(i.Side),
		LimitPrice:  gtype.Uint64(i.LimitPrice),
		MaxQuantity: gtype.Uint64(i.MaxQuantity),
		OrderType:   NewOrderType(i.OrderType),
		ClientID:    gtype.Uint64(i.ClientID),
		Accounts: SerumNewOrderAccounts{
			Market:          i.Accounts.Market.PublicKey.String(),
			OpenOrders:      i.Accounts.OpenOrders.PublicKey.String(),
			RequestQueue:    i.Accounts.RequestQueue.PublicKey.String(),
			Payer:           i.Accounts.Payer.PublicKey.String(),
			Owner:           i.Accounts.Owner.PublicKey.String(),
			CoinVault:       i.Accounts.CoinVault.PublicKey.String(),
			PcVault:         i.Accounts.PCVault.PublicKey.String(),
			SplTokenProgram: i.Accounts.SPLTokenProgram.PublicKey.String(),
			Rent:            i.Accounts.Rent.PublicKey.String(),
		},
	}
	if i.Accounts.SRMDiscountAccount != nil {
		v := i.Accounts.SRMDiscountAccount.PublicKey.String()
		s.Accounts.SRMDiscount = &v
	}

	return &SerumInstruction{
		inner: s,
	}
}

type SerumMatchOrderAccounts struct {
	Market            string
	RequestQueue      string
	EventQueue        string
	Bids              string
	Asks              string
	CoinFeeReceivable string
	PCFeeReceivable   string
}

type SerumMatchOrder struct {
	Limit    gtype.Uint64
	Accounts SerumMatchOrderAccounts
}

func NewSerumMatchOrder(i *serum.InstructionMatchOrder) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumMatchOrder{
			Limit: gtype.Uint64(i.Limit),
			Accounts: SerumMatchOrderAccounts{
				Market:            i.Accounts.Market.PublicKey.String(),
				RequestQueue:      i.Accounts.RequestQueue.PublicKey.String(),
				EventQueue:        i.Accounts.EventQueue.PublicKey.String(),
				Bids:              i.Accounts.Bids.PublicKey.String(),
				Asks:              i.Accounts.Asks.PublicKey.String(),
				CoinFeeReceivable: i.Accounts.CoinFeeReceivable.PublicKey.String(),
				PCFeeReceivable:   i.Accounts.PCFeeReceivable.PublicKey.String(),
			},
		},
	}
}

func (d *SerumInstruction) ToSerumMatchOrder() (*SerumMatchOrder, bool) {
	if v, ok := d.inner.(*SerumMatchOrder); ok {
		return v, true
	}
	return nil, false
}

type SerumConsumeEventsAccounts struct {
	OpenOrders        []string
	Market            string
	EventQueue        string
	CoinFeeReceivable string
	PCFeeReceivable   string
}
type SerumConsumeEvents struct {
	Limit    gtype.Uint64
	Accounts SerumConsumeEventsAccounts
}

func NewSerumConsumeEvents(i *serum.InstructionConsumeEvents) *SerumInstruction {
	s := &SerumConsumeEvents{
		Limit: gtype.Uint64(i.Limit),
		Accounts: SerumConsumeEventsAccounts{
			Market:            i.Accounts.Market.PublicKey.String(),
			EventQueue:        i.Accounts.EventQueue.PublicKey.String(),
			CoinFeeReceivable: i.Accounts.CoinFeeReceivable.PublicKey.String(),
			PCFeeReceivable:   i.Accounts.PCFeeReceivable.PublicKey.String(),
		},
	}

	for _, a := range i.Accounts.OpenOrders {
		s.Accounts.OpenOrders = append(s.Accounts.OpenOrders, a.PublicKey.String())

	}
	return &SerumInstruction{
		inner: s,
	}
}

func (d *SerumInstruction) ToSerumConsumeEvents() (*SerumConsumeEvents, bool) {
	if v, ok := d.inner.(*SerumConsumeEvents); ok {
		return v, true
	}
	return nil, false
}

type SerumCancelOrderAccounts struct {
	Market       string
	RequestQueue string
	Owner        string
}
type SerumCancelOrder struct {
	Side          SideType
	OrderId       string
	OpenOrders    string
	OpenOrderSlot gtype.Uint64

	Accounts SerumCancelOrderAccounts
}

func NewSerumCancelOrder(i *serum.InstructionCancelOrder) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumCancelOrder{
			Side:          newSideType(i.Side),
			OrderId:       i.OrderID.String(),
			OpenOrders:    i.OpenOrders.String(),
			OpenOrderSlot: gtype.Uint64(i.OpenOrderSlot),
			Accounts: SerumCancelOrderAccounts{
				Market:       i.Accounts.Market.PublicKey.String(),
				RequestQueue: i.Accounts.RequestQueue.PublicKey.String(),
				Owner:        i.Accounts.Owner.PublicKey.String(),
			},
		},
	}
}

func (d *SerumInstruction) ToSerumCancelOrder() (*SerumCancelOrder, bool) {
	if v, ok := d.inner.(*SerumCancelOrder); ok {
		return v, true
	}
	return nil, false
}

type SerumSettleFundsAccounts struct {
	Market           string
	OpenOrders       string
	Owner            string
	CoinVault        string
	PcVault          string
	CoinWallet       string
	PcWallet         string
	Signer           string
	SplTokenProgram  string
	ReferrerPCWallet *string
}
type SerumSettleFunds struct {
	Accounts SerumSettleFundsAccounts
}

func NewSerumSettleFunds(i *serum.InstructionSettleFunds) *SerumInstruction {
	s := &SerumSettleFunds{
		Accounts: SerumSettleFundsAccounts{
			Market:          i.Accounts.Market.PublicKey.String(),
			OpenOrders:      i.Accounts.OpenOrders.PublicKey.String(),
			Owner:           i.Accounts.Owner.PublicKey.String(),
			CoinVault:       i.Accounts.CoinVault.PublicKey.String(),
			PcVault:         i.Accounts.PCVault.PublicKey.String(),
			CoinWallet:      i.Accounts.CoinWallet.PublicKey.String(),
			PcWallet:        i.Accounts.PCWallet.PublicKey.String(),
			Signer:          i.Accounts.Signer.PublicKey.String(),
			SplTokenProgram: i.Accounts.SPLTokenProgram.PublicKey.String(),
		},
	}

	if i.Accounts.ReferrerPCWallet != nil {
		v := i.Accounts.ReferrerPCWallet.PublicKey.String()
		s.Accounts.ReferrerPCWallet = &v
	}
	return &SerumInstruction{
		inner: s,
	}
}

func (d *SerumInstruction) ToSerumSettleFunds() (*SerumSettleFunds, bool) {
	if v, ok := d.inner.(*SerumSettleFunds); ok {
		return v, true
	}
	return nil, false
}

type SerumCancelOrderByClientIdAccounts struct {
	Market       string
	OpenOrders   string
	RequestQueue string
	Owner        string
}

type SerumCancelOrderByClientId struct {
	ClientID gtype.Uint64
	Accounts SerumCancelOrderByClientIdAccounts
}

func NewSerumCancelOrderByClientId(i *serum.InstructionCancelOrderByClientId) *SerumInstruction {
	return &SerumInstruction{
		inner: &SerumCancelOrderByClientId{
			ClientID: gtype.Uint64(i.ClientID),
			Accounts: SerumCancelOrderByClientIdAccounts{
				Market:       i.Accounts.Market.PublicKey.String(),
				OpenOrders:   i.Accounts.OpenOrders.PublicKey.String(),
				RequestQueue: i.Accounts.RequestQueue.PublicKey.String(),
				Owner:        i.Accounts.Owner.PublicKey.String(),
			},
		},
	}
}

func (d *SerumInstruction) ToSerumCancelOrderByClientId() (*SerumCancelOrderByClientId, bool) {
	if v, ok := d.inner.(*SerumCancelOrderByClientId); ok {
		return v, true
	}
	return nil, false
}
