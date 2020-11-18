package resolvers

import (
	gtype "github.com/dfuse-io/dgraphql/types"
	"github.com/dfuse-io/solana-go"
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

type SerumInstructionResponse struct {
	TrxSignature string
	Instruction  *SerumInstruction
	err          error
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

type AccountMeta struct {
	am *solana.AccountMeta
}

func (a AccountMeta) PublicKey() string { return a.am.PublicKey.String() }
func (a AccountMeta) IsSigner() bool    { return a.am.IsSigner }
func (a AccountMeta) IsWritable() bool  { return a.am.IsWritable }

type SerumNewOrderAccounts struct {
	Market          AccountMeta
	OpenOrders      AccountMeta
	RequestQueue    AccountMeta
	Payer           AccountMeta
	Owner           AccountMeta
	CoinVault       AccountMeta
	PcVault         AccountMeta
	SplTokenProgram AccountMeta
	Rent            AccountMeta
	SRMDiscount     *AccountMeta
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
			Market:          AccountMeta{&i.Accounts.Market},
			OpenOrders:      AccountMeta{&i.Accounts.OpenOrders},
			RequestQueue:    AccountMeta{&i.Accounts.RequestQueue},
			Payer:           AccountMeta{&i.Accounts.Payer},
			Owner:           AccountMeta{&i.Accounts.Owner},
			CoinVault:       AccountMeta{&i.Accounts.CoinVault},
			PcVault:         AccountMeta{&i.Accounts.PCVault},
			SplTokenProgram: AccountMeta{&i.Accounts.SPLTokenProgram},
			Rent:            AccountMeta{&i.Accounts.Rent},
		},
	}
	if i.Accounts.SRMDiscountAccount != nil {
		v := AccountMeta{i.Accounts.SRMDiscountAccount}
		s.Accounts.SRMDiscount = &v
	}

	return &SerumInstruction{
		inner: s,
	}
}

type SerumMatchOrderAccounts struct {
	Market            AccountMeta
	RequestQueue      AccountMeta
	EventQueue        AccountMeta
	Bids              AccountMeta
	Asks              AccountMeta
	CoinFeeReceivable AccountMeta
	PCFeeReceivable   AccountMeta
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
				Market:            AccountMeta{&i.Accounts.Market},
				RequestQueue:      AccountMeta{&i.Accounts.RequestQueue},
				EventQueue:        AccountMeta{&i.Accounts.EventQueue},
				Bids:              AccountMeta{&i.Accounts.Bids},
				Asks:              AccountMeta{&i.Accounts.Asks},
				CoinFeeReceivable: AccountMeta{&i.Accounts.CoinFeeReceivable},
				PCFeeReceivable:   AccountMeta{&i.Accounts.PCFeeReceivable},
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
	OpenOrders        []AccountMeta
	Market            AccountMeta
	EventQueue        AccountMeta
	CoinFeeReceivable AccountMeta
	PCFeeReceivable   AccountMeta
}
type SerumConsumeEvents struct {
	Limit    gtype.Uint64
	Accounts SerumConsumeEventsAccounts
}

func NewSerumConsumeEvents(i *serum.InstructionConsumeEvents) *SerumInstruction {
	s := &SerumConsumeEvents{
		Limit: gtype.Uint64(i.Limit),
		Accounts: SerumConsumeEventsAccounts{
			Market:            AccountMeta{&i.Accounts.Market},
			EventQueue:        AccountMeta{&i.Accounts.EventQueue},
			CoinFeeReceivable: AccountMeta{&i.Accounts.CoinFeeReceivable},
			PCFeeReceivable:   AccountMeta{&i.Accounts.PCFeeReceivable},
		},
	}

	for _, a := range i.Accounts.OpenOrders {
		s.Accounts.OpenOrders = append(s.Accounts.OpenOrders, AccountMeta{&a})

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
	Market       AccountMeta
	RequestQueue AccountMeta
	Owner        AccountMeta
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
				Market:       AccountMeta{&i.Accounts.Market},
				RequestQueue: AccountMeta{&i.Accounts.RequestQueue},
				Owner:        AccountMeta{&i.Accounts.Owner},
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
	Market           AccountMeta
	OpenOrders       AccountMeta
	Owner            AccountMeta
	CoinVault        AccountMeta
	PcVault          AccountMeta
	CoinWallet       AccountMeta
	PcWallet         AccountMeta
	Signer           AccountMeta
	SplTokenProgram  AccountMeta
	ReferrerPCWallet *AccountMeta
}
type SerumSettleFunds struct {
	Accounts SerumSettleFundsAccounts
}

func NewSerumSettleFunds(i *serum.InstructionSettleFunds) *SerumInstruction {
	s := &SerumSettleFunds{
		Accounts: SerumSettleFundsAccounts{
			Market:          AccountMeta{&i.Accounts.Market},
			OpenOrders:      AccountMeta{&i.Accounts.OpenOrders},
			Owner:           AccountMeta{&i.Accounts.Owner},
			CoinVault:       AccountMeta{&i.Accounts.CoinVault},
			PcVault:         AccountMeta{&i.Accounts.PCVault},
			CoinWallet:      AccountMeta{&i.Accounts.CoinWallet},
			PcWallet:        AccountMeta{&i.Accounts.PCWallet},
			Signer:          AccountMeta{&i.Accounts.Signer},
			SplTokenProgram: AccountMeta{&i.Accounts.SPLTokenProgram},
		},
	}

	if i.Accounts.ReferrerPCWallet != nil {
		v := AccountMeta{i.Accounts.ReferrerPCWallet}
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
	Market       AccountMeta
	OpenOrders   AccountMeta
	RequestQueue AccountMeta
	Owner        AccountMeta
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
				Market:       AccountMeta{&i.Accounts.Market},
				OpenOrders:   AccountMeta{&i.Accounts.OpenOrders},
				RequestQueue: AccountMeta{&i.Accounts.RequestQueue},
				Owner:        AccountMeta{&i.Accounts.Owner},
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
