package serumhist

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	bin "github.com/streamingfast/binary"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	pbserumhist "github.com/streamingfast/sf-solana/pb/sf/solana/serumhist/v1"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/diff"
	"github.com/streamingfast/solana-go/programs/serum"
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

		oldOpenOrders, newOpenOrders, err := decodeOpenOrders(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionNewOrderV2: unable to decode open orders: %w", err)
		}

		s.addNewOrderEvent(eventRef, oldOpenOrders, newOpenOrders, v.LimitPrice, v.MaxCoinQuantity, pbserumhist.OrderType(v.OrderType))

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionNewOrderV3: unable to decode event queue: %w", err)
		}

		s.addOrderFill(eventRef, old, new)

	case *serum.InstructionCancelOrderByClientId:
		// V1 instruction pushes a Request::CancelOrder on the request queue, we need to find it and decode it
		eventRef.Market = v.Accounts.Market.PublicKey

		old, new, err := decodeRequestQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderByClientId: unable to decode event queue: %w", err)
		}

		s.addCancelledOrderViaRequestQueue(eventRef, old, new)

	case *serum.InstructionCancelOrder:
		eventRef.Market = v.Accounts.Market.PublicKey
		eventRef.OrderSeqNum = v.OrderID.SeqNum(v.Side)
		s.OrderCancelledEvents = append(s.OrderCancelledEvents, &OrderCancelled{
			Ref: *eventRef,
			InstrRef: &pbserumhist.InstructionRef{
				TrxHash:   eventRef.TrxHash,
				SlotHash:  eventRef.SlotHash,
				Timestamp: mustProtoTimestamp(eventRef.Timestamp),
			},
		})

	case *serum.InstructionCancelOrderByClientIdV2:
		// V1 instruction pushes a Event::EventOut on the event queue, we need to find it and decode it
		eventRef.Market = v.Accounts.Market.PublicKey

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderByClientIdV2: unable to decode event queue: %w", err)
		}

		s.addCancelledOrderViaEventQueue(eventRef, old, new)

	case *serum.InstructionCancelOrderV2:
		eventRef.Market = v.Accounts.Market.PublicKey
		eventRef.OrderSeqNum = v.OrderID.SeqNum(v.Side)
		s.OrderCancelledEvents = append(s.OrderCancelledEvents, &OrderCancelled{
			Ref: *eventRef,
			InstrRef: &pbserumhist.InstructionRef{
				TrxHash:   eventRef.TrxHash,
				SlotHash:  eventRef.SlotHash,
				Timestamp: mustProtoTimestamp(eventRef.Timestamp),
			},
		})
	case *serum.InstructionMatchOrder:
		eventRef.Market = v.Accounts.Market.PublicKey

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionMatchOrder: unable to decode event queue: %w", err)
		}

		s.addOrderFill(eventRef, old, new)
	case *serum.InstructionConsumeEvents:
		eventRef.Market = v.Accounts.Market.PublicKey

		old, new, err := decodeOpenOrders(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionConsumeEvents: unable to decode open orders: %w", err)
		}

		s.addClosedOrderEvent(eventRef, old, new)
	}

	return nil
}

func (s *SerumSlot) addOrderFill(eventRef *Ref, old, new *serum.EventQueue) {
	diff.Diff(old, new, diff.OnEvent(func(eventDiff diff.Event) {
		if match, _ := eventDiff.Match("Events[#]"); match {
			e := eventDiff.Element().Interface().(*serum.Event)
			switch eventDiff.Kind {
			case diff.KindAdded:
				eventRef.OrderSeqNum = e.OrderID.SeqNum(e.Side())
				if e.Flag.IsFill() {
					s.OrderFilledEvents = append(s.OrderFilledEvents, &FillEvent{
						Ref:            *eventRef,
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
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Orders[#]"); match {
			switch event.Kind {
			case diff.KindAdded:
				hasNewOrder := false
				newOrderIndex := uint32(0)

				if index, found := event.Path.SliceIndex(); found {
					hasNewOrder = true
					newOrderIndex = uint32(index)
				}

				if !hasNewOrder {
					zlog.Warn("expected to find a new order",
						zap.Reflect("event_ref", eventRef),
					)
					return
				}
				newOrder := new.GetOrder(newOrderIndex)
				eventRef.OrderSeqNum = newOrder.SeqNum()

				s.OrderNewEvents = append(s.OrderNewEvents, &NewOrder{
					Ref:    *eventRef,
					Trader: new.Owner,
					Order: &pbserumhist.Order{
						//Num:         newOrder.SeqNum(),
						Trader:      new.Owner.String(),
						Side:        pbserumhist.Side(newOrder.Side),
						LimitPrice:  limitPrice, // instruction
						MaxQuantity: maxQuantity,
						Type:        orderType,
						Fills:       nil,
						SlotHash:    eventRef.SlotHash,
						TrxId:       eventRef.TrxHash,
					},
				})
			}
		}
	}))
}

func (s *SerumSlot) addClosedOrderEvent(eventRef *Ref, old, new *serum.OpenOrders) {
	// 1. We need to Diff the OpenOrders account to retrieve the orderID
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Orders[#]"); match {
			switch event.Kind {
			case diff.KindRemoved:
				hasRemovedOrder := false
				oldOrderIndex := uint32(0)

				if index, found := event.Path.SliceIndex(); found {
					hasRemovedOrder = true
					oldOrderIndex = uint32(index)
				}

				if !hasRemovedOrder {
					zlog.Warn("expected to find an ordered closed",
						zap.Reflect("event_ref", eventRef),
					)
					return
				}
				newOrder := old.GetOrder(oldOrderIndex)
				eventRef.OrderSeqNum = newOrder.SeqNum()
				s.OrderClosedEvents = append(s.OrderClosedEvents, &OrderClosed{
					Ref: *eventRef,
					InstrRef: &pbserumhist.InstructionRef{
						TrxHash:   eventRef.TrxHash,
						SlotHash:  eventRef.SlotHash,
						Timestamp: mustProtoTimestamp(eventRef.Timestamp),
					},
				})
			}
		}
	}))
}

func (s *SerumSlot) addCancelledOrderViaEventQueue(eventRef *Ref, old, new *serum.EventQueue) {
	diff.Diff(old, new, diff.OnEvent(func(eventDiff diff.Event) {
		if match, _ := eventDiff.Match("Events[#]"); match {
			e := eventDiff.Element().Interface().(*serum.Event)
			switch eventDiff.Kind {
			case diff.KindAdded:
				eventRef.OrderSeqNum = e.OrderID.SeqNum(e.Side())
				if e.Flag.IsOut() {
					s.OrderCancelledEvents = append(s.OrderCancelledEvents, &OrderCancelled{
						Ref: *eventRef,
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

func (s *SerumSlot) addCancelledOrderViaRequestQueue(eventRef *Ref, old, new *serum.RequestQueue) {
	diff.Diff(old, new, diff.OnEvent(func(eventDiff diff.Event) {
		if match, _ := eventDiff.Match("Requests[#]"); match {
			e := eventDiff.Element().Interface().(*serum.Request)
			switch eventDiff.Kind {
			case diff.KindAdded:
				eventRef.OrderSeqNum = e.OrderID.SeqNum(e.Side())
				if e.Flag.IsCancelOrder() {
					s.OrderCancelledEvents = append(s.OrderCancelledEvents, &OrderCancelled{
						Ref: *eventRef,
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

func decodeRequestQueue(accountChanges []*pbcodec.AccountChange) (old, new *serum.RequestQueue, err error) {
	requestQueueAccountChange, err := findAccountChange(accountChanges, func(flag *serum.AccountFlag) bool {
		return flag.Is(serum.AccountFlagInitialized) && flag.Is(serum.AccountFlagRequestQueue)
	})

	if requestQueueAccountChange == nil {
		return nil, nil, nil
	}

	if err := bin.NewDecoder(requestQueueAccountChange.PrevData).Decode(&old); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'request queue' old data: %w", err)
	}

	if err := bin.NewDecoder(requestQueueAccountChange.NewData).Decode(&new); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'request queue' new data: %w", err)
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
