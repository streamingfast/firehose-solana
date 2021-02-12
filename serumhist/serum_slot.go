package serumhist

import (
	"encoding/binary"
	"encoding/hex"
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

	ordersExecuted  []*orderExecuted
	ordersCancelled []*orderCancelled
	ordersClosed    []*orderClosed
	fills           []*fill
}

func newSerumSlot() *serumSlot {
	return &serumSlot{
		tradingAccountCache: nil,
		fills:               nil,
	}
}

type serumTradingAccount struct {
	trader         solana.PublicKey
	tradingAccount solana.PublicKey
}

type fill struct {
	trader         solana.PublicKey
	fill           *pbserumhist.Fill
	slotNumber     uint64
	trxIdx         uint32
	instIdx        uint32
	tradingAccount solana.PublicKey
	market         solana.PublicKey
	orderSeqNum    uint64
}

type serumOrder struct {
	order       *pbserumhist.Order
	market      solana.PublicKey
	orderSeqNum uint64
	slotNumber  uint64
	trxIdx      uint32
	instIdx     uint32
}

type orderExecuted struct {
	market      solana.PublicKey
	orderSeqNum uint64
	slotNumber  uint64
	trxIdx      uint32
	instIdx     uint32
}

type orderClosed struct {
	market      solana.PublicKey
	orderSeqNum uint64
	slotNumber  uint64
	trxIdx      uint32
	instIdx     uint32
	instRef     *pbserumhist.InstructionRef
}

type orderCancelled struct {
	market      solana.PublicKey
	orderSeqNum uint64
	slotNumber  uint64
	trxIdx      uint32
	instIdx     uint32
	instRef     *pbserumhist.InstructionRef
}

func (s *serumSlot) processInstruction(slotNumber uint64, trxIdx, instIdx uint32, trxId, slotHash string, blkTime time.Time, instruction *serum.Instruction, accChanges []*pbcodec.AccountChange) error {
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

	case *serum.InstructionNewOrderV2:
		s.tradingAccountCache = append(s.tradingAccountCache, &serumTradingAccount{
			trader:         v.Accounts.Owner.PublicKey,
			tradingAccount: v.Accounts.OpenOrders.PublicKey,
		})

	//case *serum.InstructionCancelOrderByClientId:
	//case *serum.InstructionCancelOrder:

	case *serum.InstructionCancelOrderByClientIdV2:
		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderByClientIdV2: unable to decode event queue: %w", err)
		}

		market := v.Accounts.Market.PublicKey
		s.setSerumCancellations(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, old, new)

	case *serum.InstructionCancelOrderV2:
		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionCancelOrderV2: unable to decode event queue: %w", err)
		}

		market := v.Accounts.Market.PublicKey
		s.setSerumCancellations(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, old, new)

	case *serum.InstructionNewOrderV3:
		s.tradingAccountCache = append(s.tradingAccountCache, &serumTradingAccount{
			trader:         v.Accounts.Owner.PublicKey,
			tradingAccount: v.Accounts.OpenOrders.PublicKey,
		})

		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionNewOrderV3: unable to decode event queue: %w", err)
		}

		market := v.Accounts.Market.PublicKey
		s.setSerumFillAndOutEvent(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, old, new, true)

	case *serum.InstructionMatchOrder:
		old, new, err := decodeEventQueue(accChanges)
		if err != nil {
			return fmt.Errorf("InstructionMatchOrder: unable to decode event queue: %w", err)
		}

		market := v.Accounts.Market.PublicKey
		s.setSerumFillAndOutEvent(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, old, new, false)
	}

	return nil
}

func (s *serumSlot) setSerumCancellations(slotNumber uint64, blkTime time.Time, trxIdx, instIdx uint32, trxId, slotHash string, market solana.PublicKey, old, new *serum.EventQueue) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			switch event.Kind {
			case diff.KindAdded:
				if e.Flag.IsOut() {
					number := make([]byte, 16)
					binary.BigEndian.PutUint64(number[:], e.OrderID.Lo)
					binary.BigEndian.PutUint64(number[8:], e.OrderID.Hi)

					s.ordersCancelled = append(s.ordersCancelled, &orderCancelled{
						market:      market,
						orderSeqNum: extractOrderSeqNum(e.Side(), e.OrderID),
						slotNumber:  slotNumber,
						trxIdx:      trxIdx,
						instIdx:     instIdx,
						instRef: &pbserumhist.InstructionRef{
							SlotHash:  slotHash,
							TrxHash:   trxId,
							Timestamp: mustProtoTimestamp(blkTime),
						},
					})
				}
			}
		}
	}))
}

func (s *serumSlot) setSerumFillAndOutEvent(slotNumber uint64, blkTime time.Time, trxIdx, instIdx uint32, trxId, slotHash string, market solana.PublicKey, old, new *serum.EventQueue, processOutAsOrderExecuted bool) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			switch event.Kind {
			case diff.KindAdded:
				if e.Flag.IsFill() {
					number := make([]byte, 16)
					binary.BigEndian.PutUint64(number[:], e.OrderID.Lo)
					binary.BigEndian.PutUint64(number[8:], e.OrderID.Hi)

					s.fills = append(s.fills, &fill{
						slotNumber:     slotNumber,
						trxIdx:         trxIdx,
						instIdx:        instIdx,
						market:         market,
						tradingAccount: e.Owner,
						orderSeqNum:    extractOrderSeqNum(e.Side(), e.OrderID),
						fill: &pbserumhist.Fill{
							OrderId:           hex.EncodeToString(number[:8]) + hex.EncodeToString(number[8:]),
							Side:              pbserumhist.Side(e.Side()),
							SlotHash:          slotHash,
							TrxId:             trxId,
							Maker:             false,
							NativeQtyPaid:     e.NativeQtyPaid,
							NativeQtyReceived: e.NativeQtyReleased,
							NativeFeeOrRebate: e.NativeFeeOrRebate,
							FeeTier:           pbserumhist.FeeTier(e.FeeTier),
							Timestamp:         mustProtoTimestamp(blkTime),
						},
					})
					return
				}

				if e.Flag.IsOut() {
					number := make([]byte, 16)
					binary.BigEndian.PutUint64(number[:], e.OrderID.Lo)
					binary.BigEndian.PutUint64(number[8:], e.OrderID.Hi)

					// if the new event OUT originates from a matching order instruction, we are unable to determine whether or not
					// it is due to an order being executed or cancelled. thus, we store it as an ORDER CLOSED event and we will determine
					// whether or not it was actually executed or cancelled when we stitch the order events together
					// if the new event OUT originates from a new order v2 instruction, we know that it is a ORDER EXECUTED event
					if processOutAsOrderExecuted {
						s.ordersExecuted = append(s.ordersExecuted, &orderExecuted{
							market:      market,
							orderSeqNum: extractOrderSeqNum(e.Side(), e.OrderID),
							slotNumber:  slotNumber,
							trxIdx:      trxIdx,
							instIdx:     instIdx,
						})
						return
					}

					s.ordersClosed = append(s.ordersClosed, &orderClosed{
						market:      market,
						orderSeqNum: extractOrderSeqNum(e.Side(), e.OrderID),
						slotNumber:  slotNumber,
						trxIdx:      trxIdx,
						instIdx:     instIdx,
						instRef: &pbserumhist.InstructionRef{
							SlotHash:  slotHash,
							TrxHash:   trxId,
							Timestamp: mustProtoTimestamp(blkTime),
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
