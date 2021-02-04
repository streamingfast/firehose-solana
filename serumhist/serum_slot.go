package serumhist

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	bin "github.com/dfuse-io/binary"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/diff"
	"github.com/dfuse-io/solana-go/programs/serum"
	"go.uber.org/zap"
)

type serumSlot struct {
	tradingAccountCache []*serumTradingAccount
	fills               []*serumFill
}

func newSerumSlot() *serumSlot {
	return &serumSlot{
		tradingAccountCache: []*serumTradingAccount{},
		fills:               []*serumFill{},
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

func (s *serumSlot) processInstruction(slotNumber uint64, trxIdx uint64, instIdx uint64, blkTime time.Time, instruction *serum.Instruction, accChanges []*pbcodec.AccountChange) error {

	logFields := []zap.Field{
		zap.Uint64("slot_number", slotNumber),
		zap.Uint64("transaction_index", trxIdx),
		zap.Uint64("instruction_index", instIdx),
	}
	if newOrder, ok := instruction.Impl.(*serum.InstructionNewOrder); ok {
		if traceEnabled {
			zlog.Debug("processing new order v1", logFields...)
		}
		s.tradingAccountCache = append(s.tradingAccountCache, &serumTradingAccount{
			trader:         newOrder.Accounts.Owner.PublicKey,
			tradingAccount: newOrder.Accounts.OpenOrders.PublicKey,
		})
	} else if newOrderV2, ok := instruction.Impl.(*serum.InstructionNewOrderV2); ok {
		if traceEnabled {
			zlog.Debug("processing new order v2", logFields...)
		}
		s.tradingAccountCache = append(s.tradingAccountCache, &serumTradingAccount{
			trader:         newOrderV2.Accounts.Owner.PublicKey,
			tradingAccount: newOrderV2.Accounts.OpenOrders.PublicKey,
		})
	} else if mathOrder, ok := instruction.Impl.(*serum.InstructionMatchOrder); ok {
		if traceEnabled {
			zlog.Debug("processing match order", logFields...)
		}
		serumFills, err := s.processMatchOrderInstruction(slotNumber, blkTime, trxIdx, instIdx, mathOrder, accChanges)
		if err != nil {
			return fmt.Errorf("generating serum fills: %w", err)
		}
		s.fills = append(s.fills, serumFills...)
	}
	return nil

}

func (s *serumSlot) processMatchOrderInstruction(slotNumber uint64, blkTime time.Time, trxIdx, instIdx uint64, inst *serum.InstructionMatchOrder, accountChanges []*pbcodec.AccountChange) (out []*serumFill, err error) {
	eventQueueAccountChange, err := filterAccountChange(accountChanges, func(flag *serum.AccountFlag) bool {
		return flag.Is(serum.AccountFlagInitialized) && flag.Is(serum.AccountFlagEventQueue)
	})

	if eventQueueAccountChange == nil {
		return nil, nil
	}

	old, new, err := decodeEventQueue(eventQueueAccountChange)
	if err != nil {
		return nil, fmt.Errorf("unable to decode event queue change: %w", err)
	}

	return s.getFillKeyValues(slotNumber, blkTime, trxIdx, instIdx, inst.Accounts.Market.PublicKey, old, new), nil
}

func (i *serumSlot) getFillKeyValues(slotNumber uint64, blkTime time.Time, trxIdx, instIdx uint64, market solana.PublicKey, old, new *serum.EventQueue) (out []*serumFill) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			switch event.Kind {
			case diff.KindAdded:
				if e.Flag.IsFill() {
					number := make([]byte, 16)
					binary.BigEndian.PutUint64(number[:], e.OrderID.Lo)
					binary.BigEndian.PutUint64(number[8:], e.OrderID.Hi)

					out = append(out, &serumFill{
						slotNumber:     slotNumber,
						trxIdx:         trxIdx,
						instIdx:        instIdx,
						market:         market,
						tradingAccount: e.Owner,
						orderSeqNum:    extractOrderSeqNum(e.Side(), e.OrderID),
						fill: &pbserumhist.Fill{
							OrderId:           fmt.Sprintf("%s%s", hex.EncodeToString(number[:8]), hex.EncodeToString(number[8:])),
							Side:              pbserumhist.Side(e.Side()),
							Maker:             false,
							NativeQtyPaid:     e.NativeQtyPaid,
							NativeQtyReceived: e.NativeQtyReleased,
							NativeFeeOrRebate: e.NativeFeeOrRebate,
							FeeTier:           pbserumhist.FeeTier(e.FeeTier),
							Timestamp:         mustProtoTimestamp(blkTime),
						},
					})

				}
			default:
			}
		}
	}))
	return out
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
