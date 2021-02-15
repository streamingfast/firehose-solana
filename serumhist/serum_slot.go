package serumhist

import (
	"fmt"
	"time"

	bin "github.com/dfuse-io/binary"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/diff"
	"github.com/dfuse-io/solana-go/programs/serum"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.uber.org/zap"
)

type serumSlot struct {
	tradingAccountCache []*serumTradingAccount
	newOrders           []interface{}

	orderNewEvents       []*orderNewEvent
	orderFilledEvents    []*orderFillEvent
	orderExecutedEvents  []*orderExecutedEvent
	orderCancelledEvents []*orderCancelledEvent
	orderClosedEvents    []*orderClosedEvent
}

func newSerumSlot() *serumSlot {
	return &serumSlot{
		tradingAccountCache: nil,
		orderFilledEvents:   nil,
	}
}

type serumTradingAccount struct {
	trader         solana.PublicKey
	tradingAccount solana.PublicKey
}

type orderEventRef struct {
	market      solana.PublicKey
	orderSeqNum uint64
	slotNumber  uint64
	trxHash     string
	trxIdx      uint32
	instIdx     uint32
	slotHash    string
	timestamp   time.Time
}

type orderNewEvent struct {
	orderEventRef

	order *pbserumhist.Order
}

// serumHistWritter
func (o *orderNewEvent) WriteTo(writer) {
	writer.NewOrder()
}

type orderFillEvent struct {
	orderEventRef

	tradingAccount solana.PublicKey
	fill           *pbserumhist.Fill
}

type orderExecutedEvent struct {
	orderEventRef
}

type orderClosedEvent struct {
	orderEventRef

	instrRef *pbserumhist.InstructionRef
}

type orderCancelledEvent struct {
	orderEventRef

	instrRef *pbserumhist.InstructionRef
}

func (s *serumSlot) processInstruction(slotNumber uint64, trxIdx, instIdx uint32, trxId, slotHash string, blkTime time.Time, instruction *serum.Instruction, accChanges []*pbcodec.AccountChange) error {

	eventRef := orderEventRef{
		slotNumber: slotNumber,
		trxHash:    trxId,
		trxIdx:     trxIdx,
		instIdx:    instIdx,
		slotHash:   slotHash,
		timestamp:  blkTime,
	}

	if traceEnabled {
		zlog.Debug(fmt.Sprintf("processing instruction %T", instruction.Impl),
			zap.Uint64("slot_number", slotNumber),
			zap.Uint32("transaction_index", trxIdx),
			zap.Uint32("instruction_index", instIdx))
	}

	switch v := instruction.Impl.(type) {
	case *serum.InstructionNewOrder:
		s.tradingAccountCache = append(s.tradingAccountCache, &serumTradingAccount{
			trader:         v.Accounts.Owner.PublicKey,
			tradingAccount: v.Accounts.OpenOrders.PublicKey,
		})
		//  we need to look at a OpenOrder accounts and see an difss on the

	case *serum.InstructionNewOrderV2:
		s.tradingAccountCache = append(s.tradingAccountCache, &serumTradingAccount{
			trader:         v.Accounts.Owner.PublicKey,
			tradingAccount: v.Accounts.OpenOrders.PublicKey,
		})
		// TODO: We need to log the created event here....

	case *serum.InstructionNewOrderV3:
		s.tradingAccountCache = append(s.tradingAccountCache, &serumTradingAccount{
			trader:         v.Accounts.Owner.PublicKey,
			tradingAccount: v.Accounts.OpenOrders.PublicKey,
		})

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionNewOrderV3: unable to decode event queue: %w", err)
		}

		eventRef.market = v.Accounts.Market.PublicKey
		s.addOrderFillAndCloseEvent(eventRef, old, new, true)

	//case *serum.InstructionCancelOrderByClientId:
	//case *serum.InstructionCancelOrder:
	case *serum.InstructionCancelOrderByClientIdV2:
		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderByClientIdV2: unable to decode event queue: %w", err)
		}

		eventRef.market = v.Accounts.Market.PublicKey
		s.addOrderCancellationEvent(eventRef, old, new)

	case *serum.InstructionCancelOrderV2:
		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderV2: unable to decode event queue: %w", err)
		}

		eventRef.market = v.Accounts.Market.PublicKey
		s.addOrderCancellationEvent(eventRef, old, new)

	case *serum.InstructionMatchOrder:
		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionMatchOrder: unable to decode event queue: %w", err)
		}

		eventRef.market = v.Accounts.Market.PublicKey
		s.addOrderFillAndCloseEvent(eventRef, old, new, false)
	}

	return nil
}

func (s *serumSlot) addNewOrderEvent(eventRef orderEventRef, old, new *serum.OpenOrders) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Orders[#]"); match {
		}
	}))
}

func (s *serumSlot) addOrderCancellationEvent(eventRef orderEventRef, old, new *serum.EventQueue) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			switch event.Kind {
			case diff.KindAdded:
				if e.Flag.IsOut() {
					eventRef.orderSeqNum = e.OrderID.SeqNum(e.Side())
					s.orderCancelledEvents = append(s.orderCancelledEvents, &orderCancelledEvent{
						orderEventRef: eventRef,
						instrRef: &pbserumhist.InstructionRef{
							TrxHash:   eventRef.trxHash,
							SlotHash:  eventRef.slotHash,
							Timestamp: mustProtoTimestamp(eventRef.timestamp),
						},
					})
				}
			}
		}
	}))
}

func (s *serumSlot) addOrderFillAndCloseEvent(eventRef orderEventRef, old, new *serum.EventQueue, processOutAsOrderExecuted bool) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			switch event.Kind {
			case diff.KindAdded:
				if e.Flag.IsFill() {
					eventRef.orderSeqNum = e.OrderID.SeqNum(e.Side())
					s.orderFilledEvents = append(s.orderFilledEvents, &orderFillEvent{
						orderEventRef:  eventRef,
						tradingAccount: e.Owner,
						fill: &pbserumhist.Fill{
							OrderId:           e.OrderID.HexString(false),
							Side:              pbserumhist.Side(e.Side()),
							SlotHash:          eventRef.slotHash,
							TrxId:             eventRef.trxHash,
							Maker:             false,
							NativeQtyPaid:     e.NativeQtyPaid,
							NativeQtyReceived: e.NativeQtyReleased,
							NativeFeeOrRebate: e.NativeFeeOrRebate,
							FeeTier:           pbserumhist.FeeTier(e.FeeTier),
							Timestamp:         mustProtoTimestamp(eventRef.timestamp),
						},
					})
					return
				}

				if e.Flag.IsOut() {
					// if the new event OUT originates from a matching order instruction, we are unable to determine whether or not
					// it is due to an order being executed or cancelled. thus, we store it as an ORDER CLOSED event and we will determine
					// whether or not it was actually executed or cancelled when we stitch the order events together
					// if the new event OUT originates from a new order v2 instruction, we know that it is a ORDER EXECUTED event
					if processOutAsOrderExecuted {
						s.orderExecutedEvents = append(s.orderExecutedEvents, &orderExecutedEvent{
							orderEventRef: eventRef,
						})
						return
					}

					s.orderClosedEvents = append(s.orderClosedEvents, &orderClosedEvent{
						orderEventRef: eventRef,
						instrRef: &pbserumhist.InstructionRef{
							TrxHash:   eventRef.trxHash,
							SlotHash:  eventRef.slotHash,
							Timestamp: mustProtoTimestamp(eventRef.timestamp),
						},
					})
					return
				}
			}
		}
	}))
}

func decodeEventQueue(accountChanges []*pbcodec.AccountChange) (old, new *serum.EventQueue, err error) {
	eventQueueAccountChange, err := findAccountChange(accountChanges, func(flag *serum.AccountFlag) bool {
		return flag.Is(serum.AccountFlagInitialized) && flag.Is(serum.AccountFlagEventQueue)
	})

	if eventQueueAccountChange == nil {
		return nil, nil, nil
	}

	if err := bin.NewDecoder(eventQueueAccountChange.PrevData).Decode(&old); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'event queue' old data: %w", err)
	}

	if err := bin.NewDecoder(eventQueueAccountChange.NewData).Decode(&new); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'event queue' new data: %w", err)
	}

	return
}

func extractOrderSeqNum(side serum.Side, orderID bin.Uint128) uint64 {
	if side == serum.SideBid {
		return ^orderID.Lo
	}
	return orderID.Lo
}

func mustProtoTimestamp(in time.Time) *timestamp.Timestamp {
	out, err := ptypes.TimestampProto(in)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp conversion %q: %s", in, err))
	}
	return out
}
