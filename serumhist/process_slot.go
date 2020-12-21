package serumhist

import (
	"fmt"
	pbserum "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serum/v1"

	"github.com/dfuse-io/solana-go"

	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"

	"github.com/dfuse-io/solana-go/diff"

	bin "github.com/dfuse-io/binary"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	kvdb "github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go/programs/serum"
	"go.uber.org/zap"
)

func (l *Injector) ProcessSlot(slot *pbcodec.Slot) error {
	if traceEnabled {
		zlog.Debug("processing slot", zap.String("slot_id", slot.Id))
	}

	if err := l.processSerumSlot(slot); err != nil {
		return fmt.Errorf("put slot: unable to process serum slot: %w", err)
	}

	return nil
}

func (l *Injector) processSerumSlot(slot *pbcodec.Slot) error {
	for _, transaction := range slot.Transactions {
		for idx, instruction := range transaction.Instructions {
			if instruction.ProgramId != serum.PROGRAM_ID.String() {
				if traceEnabled {
					zlog.Debug("skipping non-serum instruction",
						zap.String("transaction_id", transaction.Id),
						zap.Int("instruction_index", idx),
						zap.String("program_id", instruction.ProgramId),
					)
				}
				continue
			}

			var serumInstruction *serum.Instruction
			if err := bin.NewDecoder(instruction.Data).Decode(&serumInstruction); err != nil {
				zlog.Warn("unable to decode serum instruction skipping",
					zap.String("transaction_id", transaction.Id),
					zap.Int("instruction_index", idx),
				)
				continue
			}

			// we only care about new order instruction that modify the request queue
			if newOrder, ok := serumInstruction.Impl.(*serum.InstructionNewOrder); ok {
				processNewOrderRequestQueue(slot.Number, newOrder, instruction.AccountChanges)
				continue
			}

			// we only care about new order instruction that modify the event queue
			if mathOrder, ok := serumInstruction.Impl.(*serum.InstructionMatchOrder); ok {
				processMatchOrderEventQueue(slot.Number, mathOrder, instruction.accountChange)
				continue
			}

		}
	}
	return nil
}

func getAccountChange(accountChanges []*pbcodec.AccountChange, filter func(f *serum.AccountFlag) bool) (*pbcodec.AccountChange, error) {
	for _, accountChange := range accountChanges {
		var f *serum.AccountFlag
		if err := bin.NewDecoder(accountChange.PrevData).Decode(&f); err != nil {
			return nil, fmt.Errorf("unable to deocde account flag: %w", err)
		}

		if filter(f) {
			return accountChange, nil
		}
	}
	return nil, nil
}

func processNewOrderRequestQueue(slotNumber uint64, inst *serum.InstructionNewOrder, accountChanges []*pbcodec.AccountChange) (out []*kvdb.KV, err error) {
	requestQueueAccountChange, err := getAccountChange(accountChanges, func(f *serum.AccountFlag) bool {
		return f.Is(serum.AccountFlagInitialized) && f.Is(serum.AccountFlagRequestQueue)
	})

	if requestQueueAccountChange == nil {
		return nil, fmt.Errorf("unable to retrieve Request Queue Account: %w", err)
	}

	old, new, err := decodeRequestQueue(requestQueueAccountChange)
	if err != nil {
		return nil, fmt.Errorf("unable to decode request queue change: %w", err)
	}

	return generateNewOrderKeys(slotNumber, inst.Side, inst.Accounts.Owner.PublicKey, inst.Accounts.Market.PublicKey, old, new), nil
}

func generateNewOrderKeys(slotNumber uint64, side serum.Side, owner, market solana.PublicKey, old *serum.RequestQueue, new *serum.RequestQueue) (out []*kvdb.KV) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Requests[#]"); match {
			request := event.Element().Interface().(*serum.Request)
			orderSeqNum := extractOrderSeqNum(side, request.OrderID)
			switch event.Kind {
			case diff.KindChanged:
				// this is probably a partial fill we don't care about this right now
			case diff.KindAdded:
				// etiehr a cancel request or ad new order
				// this should create keys
				switch request.RequestFlags {

				case serum.RequestFlagNewOrder:
					out = append(out, &kvdb.KV{
						Key:   keyer.EncodeOrdersByMarketPubkey(owner, market, orderSeqNum, slotNumber),
						Value: nil,
					})
					out = append(out, &kvdb.KV{
						Key:   keyer.EncodeOrdersByPubkey(owner, market, orderSeqNum, slotNumber),
						Value: nil,
					})
				case serum.RequestFlagCancelOrder:
				}
			case diff.KindRemoved:
				// this is a request that for either canceled or fully filled
			default:
			}
		}
	}))
	return out
}

func processMatchOrderEventQueue(slotNumber uint64, inst *serum.InstructionMatchOrder, accountChanges []*pbcodec.AccountChange) (out []*kvdb.KV, err error) {
	eventQueueAccountChange, err := getAccountChange(accountChanges, func(f *serum.AccountFlag) bool {
		return f.Is(serum.AccountFlagInitialized) && f.Is(serum.AccountFlagEventQueue)
	})

	if eventQueueAccountChange == nil {
		return nil, fmt.Errorf("unable to Event Queue Account: %w", err)
	}

	old, new, err := decodeEventQueue(eventQueueAccountChange)
	if err != nil {
		return nil, fmt.Errorf("unable to decode event queue change: %w", err)
	}

	return generateFillKeys(slotNumber, inst.Accounts.Market.PublicKey, old, new), nil
}

//Flag              EventFlag
//OwnerSlot         uint8
//FeeTier           uint8
//Padding           [5]uint8
//NativeQtyReleased uint64
//NativeQtyPaid     uint64
//NativeFeeOrRebate uint64
//OrderID           bin.Uint128
//Owner             solana.PublicKey
//ClientOrderID     uint64

func generateFillKeys(slotNumber uint64, side serum.Side, owner, market solana.PublicKey, old *serum.EventQueue, new *serum.EventQueue) (out []*kvdb.KV) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			orderSeqNum := extractOrderSeqNum(side, e.OrderID)
			switch event.Kind {
			case diff.KindChanged:
				// this is probably a partial fill we don't care about this right now
			case diff.KindAdded:
				// etiehr a cancel request or ad new order
				// this should create keys
				switch e.Flag {
				//	case serum.EventFlagOut:
				//	case serum.EventFlagBid:
				//	case serum.EventFlagMaker:
				case serum.EventFlagFill:
					OrderID

					fill := pbserum.Fill{
						Pubkey:               owner,
						OrderId:              ,
						IsAsk:                false,
						Maker:                false,
						NativeQtyPaid:        0,
						NativeQtyReceived:    0,
						NativeFeeOrRebate:    0,
						FeeTier:              "",
					}

					out = append(out, &kvdb.KV{
						Key:   keyer.EncodeOrdersByMarketPubkey(owner, market, orderSeqNum, slotNumber),
						Value: ,
					})
					out = append(out, &kvdb.KV{
						Key:   keyer.EncodeOrdersByPubkey(owner, market, orderSeqNum, slotNumber),
						Value: nil,
					})
				case serum.RequestFlagCancelOrder:
				}
			case diff.KindRemoved:
				// this is a request that for either canceled or fully filled
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

func decodeOpenOrder(accountChange *pbcodec.AccountChange) (old *serum.OpenOrdersV2, new *serum.OpenOrdersV2, err error) {
	if err := bin.NewDecoder(accountChange.PrevData).Decode(&old); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'open orders' old data: %w", err)
	}

	if err := bin.NewDecoder(accountChange.NewData).Decode(&new); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'open orders' new data: %w", err)
	}
	return
}

func decodeRequestQueue(accountChange *pbcodec.AccountChange) (old *serum.RequestQueue, new *serum.RequestQueue, err error) {
	if err := bin.NewDecoder(accountChange.PrevData).Decode(&old); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'request queue' old data: %w", err)
	}

	if err := bin.NewDecoder(accountChange.NewData).Decode(&new); err != nil {
		return nil, nil, fmt.Errorf("unable to decode 'request queue' new data: %w", err)
	}

	return
}
