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
		s.TradingAccountCache = append(s.TradingAccountCache, &serumTradingAccount{
			Trader:         v.Accounts.Owner.PublicKey,
			TradingAccount: v.Accounts.OpenOrders.PublicKey,
		})
		//  we need to look at a OpenOrder accounts and see an difss on the

	case *serum.InstructionNewOrderV2:
		s.TradingAccountCache = append(s.TradingAccountCache, &serumTradingAccount{
			Trader:         v.Accounts.Owner.PublicKey,
			TradingAccount: v.Accounts.OpenOrders.PublicKey,
		})
		// TODO: We need to log the created event here....

	case *serum.InstructionNewOrderV3:
		s.TradingAccountCache = append(s.TradingAccountCache, &serumTradingAccount{
			Trader:         v.Accounts.Owner.PublicKey,
			TradingAccount: v.Accounts.OpenOrders.PublicKey,
		})

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionNewOrderV3: unable to decode event queue: %w", err)
		}

		eventRef.Market = v.Accounts.Market.PublicKey
		s.addOrderFillAndCloseEvent(eventRef, old, new, true)

	//case *serum.InstructionCancelOrderByClientId:
	//case *serum.InstructionCancelOrder:
	case *serum.InstructionCancelOrderByClientIdV2:
		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderByClientIdV2: unable to decode event queue: %w", err)
		}

		eventRef.Market = v.Accounts.Market.PublicKey
		s.addOrderCancellationEvent(eventRef, old, new)

	case *serum.InstructionCancelOrderV2:
		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderV2: unable to decode event queue: %w", err)
		}

		eventRef.Market = v.Accounts.Market.PublicKey
		s.addOrderCancellationEvent(eventRef, old, new)

	case *serum.InstructionMatchOrder:
		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionMatchOrder: unable to decode event queue: %w", err)
		}

		eventRef.Market = v.Accounts.Market.PublicKey
		s.addOrderFillAndCloseEvent(eventRef, old, new, false)
	}

	return nil
}

func (s *SerumSlot) addNewOrderEvent(eventRef Ref, old, new *serum.OpenOrders) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Orders[#]"); match {
		}
	}))
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

				//if e.Flag.IsOut() {
				//	// if the new event OUT originates from a matching order instruction, we are unable to determine whether or not
				//	// it is due to an order being executed or cancelled. thus, we store it as an ORDER CLOSED event and we will determine
				//	// whether or not it was actually executed or cancelled when we stitch the order events together
				//	// if the new event OUT originates from a new order v2 instruction, we know that it is a ORDER EXECUTED event
				//	if processOutAsOrderExecuted {
				//		s.OrderExecutedEvents = append(s.OrderExecutedEvents, &OrderExecuted{
				//			Ref: eventRef,
				//		})
				//		return
				//	}
				//
				//	s.OrderClosedEvents = append(s.OrderClosedEvents, &OrderClosed{
				//		Ref: eventRef,
				//		InstrRef: &pbserumhist.InstructionRef{
				//			TrxHash:   eventRef.TrxHash,
				//			SlotHash:  eventRef.SlotHash,
				//			Timestamp: mustProtoTimestamp(eventRef.Timestamp),
				//		},
				//	})
				//	return
				//}
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
