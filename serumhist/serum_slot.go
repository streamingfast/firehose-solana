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

	orderExecuted      []*serumOrderExecuted
	orderCancellations []*serumOrderCancelled
	fills              []*serumFill
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

type serumFill struct {
	trader         solana.PublicKey
	fill           *pbserumhist.Fill
	slotNumber     uint64
	trxIdx         uint64
	instIdx        uint64
	tradingAccount solana.PublicKey
	market         solana.PublicKey
	orderSeqNum    uint64
}

type serumOrder struct {
	order       *pbserumhist.Order
	market      solana.PublicKey
	orderSeqNum uint64
	slotNumber  uint64
	trxIdx      uint64
	instIdx     uint64
}

type serumOrderExecuted struct {
	market      solana.PublicKey
	orderSeqNum uint64
	slotNumber  uint64
	trxIdx      uint64
	instIdx     uint64
}

type serumOrderCancelled struct {
	market      solana.PublicKey
	orderSeqNum uint64
	slotNumber  uint64
	trxIdx      uint64
	instIdx     uint64
}

func (s *serumSlot) processInstruction(slotNumber uint64, trxIdx uint64, instIdx uint64, trxId, slotHash string, blkTime time.Time, instruction *serum.Instruction, accChanges []*pbcodec.AccountChange) error {
	if traceEnabled {
		zlog.Debug(fmt.Sprintf("processing instruction %T", instruction.Impl),
			zap.Uint64("slot_number", slotNumber),
			zap.Uint64("transaction_index", trxIdx),
			zap.Uint64("instruction_index", instIdx))
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
		market := v.Accounts.Market.PublicKey
		if err := s.extractCancellationsFromInstruction(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, accChanges); err != nil{
			return fmt.Errorf("generating instruction cancel orders by client v2: %w", err)
		}

	case *serum.InstructionCancelOrderV2:
		market := v.Accounts.Market.PublicKey
		if err := s.extractCancellationsFromInstruction(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, accChanges); err != nil{
			return fmt.Errorf("generating instruction cancel orders v2: %w", err)
		}

	case *serum.InstructionNewOrderV3:
		s.tradingAccountCache = append(s.tradingAccountCache, &serumTradingAccount{
			trader:         v.Accounts.Owner.PublicKey,
			tradingAccount: v.Accounts.OpenOrders.PublicKey,
		})

		market := v.Accounts.Market.PublicKey
		if err := s.extractFillsFromInstruction(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, accChanges); err != nil {
			return fmt.Errorf("generating serum fills from new order v3: %w", err)
		}

	case *serum.InstructionMatchOrder:
		market := v.Accounts.Market.PublicKey
		if err := s.extractFillsFromInstruction(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, accChanges); err != nil {
			return fmt.Errorf("generating serum fills from matching order: %w", err)
		}
	}

	return nil
}

func (s serumSlot) extractCancellationsFromInstruction(
	slotNumber uint64,
	blkTime time.Time,
	trxIdx uint64,
	instIdx uint64,
	trxId string,
	slotHash string,
	market solana.PublicKey,
	accountChanges []*pbcodec.AccountChange,
) (err error) {
	eventQueueAccountChange, err := findAccountChange(accountChanges, func(flag *serum.AccountFlag) bool {
		return flag.Is(serum.AccountFlagInitialized) && flag.Is(serum.AccountFlagEventQueue)
	})

	if eventQueueAccountChange == nil {
		return nil
	}

	old, new, err := decodeEventQueue(eventQueueAccountChange)
	if err != nil {
		return fmt.Errorf("unable to decode event queue change: %w", err)
	}

	s.getCancellationKeyValues(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, old, new)
	return nil
}

func (s *serumSlot) getCancellationKeyValues(slotNumber uint64, blkTime time.Time, trxIdx, instIdx uint64, trxId, slotHash string, market solana.PublicKey, old, new *serum.EventQueue) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			switch event.Kind {
			case diff.KindAdded:
				if e.Flag.IsOut() {
					number := make([]byte, 16)
					binary.BigEndian.PutUint64(number[:], e.OrderID.Lo)
					binary.BigEndian.PutUint64(number[8:], e.OrderID.Hi)

					s.orderCancellations = append(s.orderCancellations, &serumOrderCancelled{
						market:      solana.PublicKey{},
						orderSeqNum: extractOrderSeqNum(e.Side(), e.OrderID),
						slotNumber:  slotNumber,
						trxIdx:      trxIdx,
						instIdx:     instIdx,
					})
				}
			}
		}
	}))
}

func (s *serumSlot) extractFillsFromInstruction(
	slotNumber uint64,
	blkTime time.Time,
	trxIdx uint64,
	instIdx uint64,
	trxId string,
	slotHash string,
	market solana.PublicKey,
	accountChanges []*pbcodec.AccountChange,
) (err error) {
	eventQueueAccountChange, err := findAccountChange(accountChanges, func(flag *serum.AccountFlag) bool {
		return flag.Is(serum.AccountFlagInitialized) && flag.Is(serum.AccountFlagEventQueue)
	})

	if eventQueueAccountChange == nil {
		return nil
	}

	old, new, err := decodeEventQueue(eventQueueAccountChange)
	if err != nil {
		return fmt.Errorf("unable to decode event queue change: %w", err)
	}

	s.getFillKeyValues(slotNumber, blkTime, trxIdx, instIdx, trxId, slotHash, market, old, new)
	return nil
}

func (s *serumSlot) getFillKeyValues(slotNumber uint64, blkTime time.Time, trxIdx, instIdx uint64, trxId, slotHash string, market solana.PublicKey, old, new *serum.EventQueue) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			switch event.Kind {
			case diff.KindAdded:
				if e.Flag.IsFill() {
					number := make([]byte, 16)
					binary.BigEndian.PutUint64(number[:], e.OrderID.Lo)
					binary.BigEndian.PutUint64(number[8:], e.OrderID.Hi)

					s.fills = append(s.fills, &serumFill{
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
				} else if e.Flag.IsOut() {
					number := make([]byte, 16)
					binary.BigEndian.PutUint64(number[:], e.OrderID.Lo)
					binary.BigEndian.PutUint64(number[8:], e.OrderID.Hi)

					s.orderExecuted = append(s.orderExecuted, &serumOrderExecuted{
						market:      solana.PublicKey{},
						orderSeqNum: extractOrderSeqNum(e.Side(), e.OrderID),
						slotNumber:  slotNumber,
						trxIdx:      trxIdx,
						instIdx:     instIdx,
					})
				}
			}
		}
	}))
}

func extractOrderSeqNum(side serum.Side, orderID bin.Uint128) uint64 {
	if side == serum.SideBid {
		return ^orderID.Lo
	}
	return orderID.Lo
}

func decodeEventQueue(accountChange *pbcodec.AccountChange) (old *serum.EventQueue, new *serum.EventQueue, err error) {
	if err := bin.NewDecoder(accountChange.PrevData).Decode(&old); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'event queue' old data: %w", err)
	}

	if err := bin.NewDecoder(accountChange.NewData).Decode(&new); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'event queue' new data: %w", err)
	}

	return
}

func mustProtoTimestamp(in time.Time) *timestamp.Timestamp {
	out, err := ptypes.TimestampProto(in)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp conversion %q: %s", in, err))
	}
	return out
}
