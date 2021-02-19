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

type SerumSlot struct {
	TradingAccountCache []*serumTradingAccount

	OrderNewEvents       []*NewOrder
	OrderFilledEvents    []*FillEvent
	OrderExecutedEvents  []*OrderExecuted
	OrderCancelledEvents []*OrderCancelled
	OrderClosedEvents    []*OrderClosed
}

func newSerumSlot() *SerumSlot {
	return &SerumSlot{
		TradingAccountCache: nil,
		OrderFilledEvents:   nil,
	}
}

type serumTradingAccount struct {
	Trader         solana.PublicKey
	TradingAccount solana.PublicKey
}

func (s *SerumSlot) processInstruction(slotNumber uint64, trxIdx, instIdx uint32, trxId, slotHash string, blkTime time.Time, instruction *serum.Instruction, accChanges []*pbcodec.AccountChange) error {

	eventRef := &Ref{
		SlotNumber: slotNumber,
		TrxHash:    trxId,
		TrxIdx:     trxIdx,
		InstIdx:    instIdx,
		SlotHash:   slotHash,
		Timestamp:  blkTime,
	}

	if traceEnabled {
		zlog.Debug(fmt.Sprintf("processing instruction %T", instruction.Impl),
			zap.Uint64("slot_number", slotNumber),
			zap.Uint32("transaction_index", trxIdx),
			zap.Uint32("instruction_index", instIdx))
	}

	switch v := instruction.Impl.(type) {
	case *serum.InstructionNewOrder:
		eventRef.Market = v.Accounts.Market.PublicKey
		s.TradingAccountCache = append(s.TradingAccountCache, &serumTradingAccount{
			Trader:         v.Accounts.Owner.PublicKey,
			TradingAccount: v.Accounts.OpenOrders.PublicKey,
		})

		old, new, err := decodeOpenOrders(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionNewOrder: unable to decode open orders: %w", err)
		}

		s.addNewOrderEvent(eventRef, old, new, v.LimitPrice, v.MaxQuantity, pbserumhist.OrderType(v.OrderType))

	case *serum.InstructionNewOrderV2:
		eventRef.Market = v.Accounts.Market.PublicKey
		s.TradingAccountCache = append(s.TradingAccountCache, &serumTradingAccount{
			Trader:         v.Accounts.Owner.PublicKey,
			TradingAccount: v.Accounts.OpenOrders.PublicKey,
		})

		old, new, err := decodeOpenOrders(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionNewOrderV2: unable to decode open orders: %w", err)
		}

		s.addNewOrderEvent(eventRef, old, new, v.LimitPrice, v.MaxQuantity, pbserumhist.OrderType(v.OrderType))

	case *serum.InstructionNewOrderV3:
		eventRef.Market = v.Accounts.Market.PublicKey

		s.TradingAccountCache = append(s.TradingAccountCache, &serumTradingAccount{
			Trader:         v.Accounts.Owner.PublicKey,
			TradingAccount: v.Accounts.OpenOrders.PublicKey,
		})

		//oldOO, newOO, err := decodeOpenOrders(accChanges)
		//if err != nil {
		//	return fmt.Errorf("InstructionNewOrderV2: unable to decode open orders: %w", err)
		//}

		//s.addNewOrderEvent(eventRef, old, new, v.LimitPrice, v, pbserumhist.OrderType(v.OrderType))

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionNewOrderV3: unable to decode event queue: %w", err)
		}

		s.addOrderFillAndCloseEvent(eventRef, old, new, true)

	//case *serum.InstructionCancelOrderByClientId:
	//case *serum.InstructionCancelOrder:
	case *serum.InstructionCancelOrderByClientIdV2:
		eventRef.Market = v.Accounts.Market.PublicKey

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderByClientIdV2: unable to decode event queue: %w", err)
		}

		s.addOrderCancellationEvent(eventRef, old, new)
	case *serum.InstructionCancelOrderV2:
		eventRef.Market = v.Accounts.Market.PublicKey

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderV2: unable to decode event queue: %w", err)
		}

		s.addOrderCancellationEvent(eventRef, old, new)
	case *serum.InstructionMatchOrder:
		eventRef.Market = v.Accounts.Market.PublicKey

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionMatchOrder: unable to decode event queue: %w", err)
		}

		s.addOrderFillAndCloseEvent(eventRef, old, new, false)
	}

	return nil
}

func (s *SerumSlot) addOrderCancellationEvent(eventRef *Ref, old, new *serum.EventQueue) {
	diff.Diff(old, new, diff.OnEvent(func(eventdiff diff.Event) {
		if match, _ := eventdiff.Match("Events[#]"); match {
			e := eventdiff.Element().Interface().(*serum.Event)
			switch eventdiff.Kind {
			case diff.KindAdded:
				if e.Flag.IsOut() {
					eventRef.OrderSeqNum = e.OrderID.SeqNum(e.Side())
					s.OrderCancelledEvents = append(s.OrderCancelledEvents, &OrderCancelled{
						Ref: eventRef,
						InstrRef: &pbserumhist.InstructionRef{
							TrxHash:   eventRef.TrxHash,
							SlotHash:  eventRef.SlotHash,
							Timestamp: mustProtoTimestamp(eventRef.Timestamp),
						},
					})
				}
			}
		}
	}))
}

func (s *SerumSlot) addOrderFillAndCloseEvent(eventRef *Ref, old, new *serum.EventQueue, processOutAsOrderExecuted bool) {
	diff.Diff(old, new, diff.OnEvent(func(eventDiff diff.Event) {
		if match, _ := eventDiff.Match("Events[#]"); match {
			e := eventDiff.Element().Interface().(*serum.Event)
			switch eventDiff.Kind {
			case diff.KindAdded:
				eventRef.OrderSeqNum = e.OrderID.SeqNum(e.Side())
				if e.Flag.IsFill() {
					s.OrderFilledEvents = append(s.OrderFilledEvents, &FillEvent{
						Ref:            eventRef,
						TradingAccount: e.Owner,
						Fill: &pbserumhist.Fill{
							OrderId:           e.OrderID.HexString(false),
							Side:              pbserumhist.Side(e.Side()),
							SlotHash:          eventRef.SlotHash,
							TrxId:             eventRef.TrxHash,
							Maker:             false,
							NativeQtyPaid:     e.NativeQtyPaid,
							NativeQtyReceived: e.NativeQtyReleased,
							NativeFeeOrRebate: e.NativeFeeOrRebate,
							FeeTier:           pbserumhist.FeeTier(e.FeeTier),
							Timestamp:         mustProtoTimestamp(eventRef.Timestamp),
						},
					})
					return
				}
			}
		}
	}))
}

func (s *SerumSlot) addNewOrderEvent(eventRef *Ref, old, new *serum.OpenOrders, limitPrice uint64, maxQuantity uint64, orderType pbserumhist.OrderType) {
	// 1. We need to Diff the OpenOrders account to retrieve the orderID
	hasNewOrder := false
	newOrderIndex := uint32(0)
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Orders[#]"); match {
			switch event.Kind {
			case diff.KindAdded:
				if index, found := event.Path.SliceIndex(); found {
					hasNewOrder = true
					newOrderIndex = uint32(index)
				}
			}
		}
	}))
	if !hasNewOrder {
		zlog.Warn("expected to find a new order",
			zap.Reflect("event_ref", eventRef),
		)
	}
	newOrder := new.GetOrder(newOrderIndex)

	s.OrderNewEvents = append(s.OrderNewEvents, &NewOrder{
		Ref: eventRef,
		Order: &pbserumhist.Order{
			Num:         newOrder.SeqNum(),
			Trader:      new.Owner.String(),
			Side:        pbserumhist.Side(newOrder.Side),
			LimitPrice:  limitPrice, // instruction
			MaxQuantity: maxQuantity,
			Type:        orderType,
			SlotHash:    eventRef.SlotHash,
			TrxId:       eventRef.TrxHash,
		},
	})
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

func decodeOpenOrders(accountChanges []*pbcodec.AccountChange) (old, new *serum.OpenOrders, err error) {
	openOrdersAccountChange, err := findAccountChange(accountChanges, func(flag *serum.AccountFlag) bool {
		return flag.Is(serum.AccountFlagInitialized) && flag.Is(serum.AccountFlagOpenOrders)
	})

	if openOrdersAccountChange == nil {
		return nil, nil, nil
	}

	if err := bin.NewDecoder(openOrdersAccountChange.PrevData).Decode(&old); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'open orders' old data: %w", err)
	}

	if err := bin.NewDecoder(openOrdersAccountChange.NewData).Decode(&new); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'open orders' new data: %w", err)
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
