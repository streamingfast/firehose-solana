package serumhist

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/golang/protobuf/proto"

	bin "github.com/dfuse-io/binary"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	kvdb "github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/diff"
	"github.com/dfuse-io/solana-go/programs/serum"
	"go.uber.org/zap"
)

func (i *Injector) ProcessSlot(ctx context.Context, slot *pbcodec.Slot) error {
	if traceEnabled {
		zlog.Debug("processing slot", zap.String("slot_id", slot.Id))
	}

	if err := i.processSerumSlot(ctx, slot); err != nil {
		return fmt.Errorf("put slot: unable to process serum slot: %w", err)
	}

	return nil
}

func (i *Injector) processSerumSlot(ctx context.Context, slot *pbcodec.Slot) error {
	for _, transaction := range slot.Transactions {
		for idx, instruction := range transaction.Instructions {
			if instruction.ProgramId != serum.PROGRAM_ID.String() {
				if traceEnabled {
					zlog.Debug("skipping non-serum instruction",
						zap.Uint64("slot_number", slot.Number),
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
					zap.Uint64("slot_number", slot.Number),
					zap.String("transaction_id", transaction.Id),
					zap.Int("instruction_index", idx),
				)
				continue
			}

			zlog.Info("processing serum instruction",
				zap.Uint64("slot_number", slot.Number),
				zap.Int("instruction_index", idx),
				zap.String("transaction_id", transaction.Id),
				zap.Uint32("serum_instruction_variant_index", serumInstruction.TypeID),
			)

			var out []*kvdb.KV
			var err error

			instructionAccountIndexes := instructionAccountIndexes(transaction.AccountKeys, instruction.AccountKeys)
			accounts, err := transaction.AccountMetaList()
			if err != nil {
				return fmt.Errorf("process serum slot: get trx account meta list: %w", err)
			}

			// we only care about new order instruction that modify the request queue
			if newOrder, ok := serumInstruction.Impl.(*serum.InstructionNewOrder); ok {
				err := newOrder.SetAccounts(accounts, instructionAccountIndexes)
				if err != nil {
					return fmt.Errorf("process serum slot: match order: set account metas: %w", err)
				}
				zlog.Info("processing new order")
				out, err = processNewOrderRequestQueue(slot.Number, newOrder, instruction.AccountChanges)
				if err != nil {
					zlog.Warn("error processing new order",
						zap.Uint64("slot_number", slot.Number),
						zap.String("error", err.Error()),
					)
					continue
				}

			}

			// we only care about new order instruction that modify the event queue
			if mathOrder, ok := serumInstruction.Impl.(*serum.InstructionMatchOrder); ok {
				err := mathOrder.SetAccounts(accounts, instructionAccountIndexes)
				if err != nil {
					return fmt.Errorf("process serum slot: match order: set account metas: %w", err)
				}
				zlog.Info("processing match order")
				out, err = processMatchOrderEventQueue(slot.Number, mathOrder, instruction.AccountChanges)
				if err != nil {
					zlog.Warn("error matching order and event queue",
						zap.Uint64("slot_number", slot.Number),
						zap.String("error", err.Error()),
					)
					continue
				}
			}

			for _, kv := range out {
				if err := i.kvdb.Put(ctx, kv.Key, kv.Value); err != nil {
					zlog.Warn("failed to write key-value", zap.Error(err))
				}
			}
		}
	}
	return nil
}

func instructionAccountIndexes(trxAccounts []string, instructionAccounts []string) []uint8 {
	var out []uint8
	for _, ia := range instructionAccounts {
		for i, ta := range trxAccounts {
			if ta == ia {
				out = append(out, uint8(i))
			}
		}
	}
	return out
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
	eventQueueAccountChange, err := getAccountChange(accountChanges, func(flag *serum.AccountFlag) bool {
		zlog.Debug("checking account change flags", zap.Stringer("flag", flag))
		return flag.Is(serum.AccountFlagInitialized) && flag.Is(serum.AccountFlagEventQueue)
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

func generateFillKeys(slotNumber uint64, market solana.PublicKey, old *serum.EventQueue, new *serum.EventQueue) (out []*kvdb.KV) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			switch event.Kind {
			case diff.KindChanged:
				// this is probably a partial fill we don't care about this right now
			case diff.KindAdded:
				fmt.Println("flag: ", e.Flag)
				if e.Flag.IsFill() {
					size := 16
					buf := make([]byte, size)
					binary.LittleEndian.PutUint64(buf, e.OrderID.Lo)
					binary.LittleEndian.PutUint64(buf[(size/2):], e.OrderID.Hi)
					fill := &pbserumhist.Fill{
						Trader:            e.Owner[:],
						OrderId:           buf,
						Side:              pbserumhist.Side(e.Side()),
						Maker:             false,
						NativeQtyPaid:     e.NativeQtyPaid,
						NativeQtyReceived: e.NativeQtyReleased,
						NativeFeeOrRebate: e.NativeFeeOrRebate,
						FeeTier:           pbserumhist.FeeTier(e.FeeTier),
					}

					if e.Side() == serum.SideAsk {
						fill.Side = 1
					}

					cnt, err := proto.Marshal(fill)
					if err != nil {
						zlog.Error("unable to marshal to fill", zap.Error(err))
						return
					}
					orderSeqNum := extractOrderSeqNum(e.Side(), e.OrderID)

					out = append(out, &kvdb.KV{
						Key:   keyer.EncodeFillData(market, orderSeqNum, slotNumber),
						Value: cnt,
					})
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
	defer func() {
		if err := recover(); err != nil {
			zlog.Error("decoder panic recover", zap.String("data", hex.EncodeToString(accountChange.PrevData)))
			panic(err)
		}
	}()

	if err := bin.NewDecoder(accountChange.PrevData).Decode(&old); err != nil {
		zlog.Warn("unable to decode 'event queue' old data", zap.String("data", hex.EncodeToString(accountChange.PrevData)))
		return nil, nil, fmt.Errorf("unable to decode 'event queue' old data: %w", err)
	}

	if err := bin.NewDecoder(accountChange.NewData).Decode(&new); err != nil {
		zlog.Warn("unable to decode 'event queue' new data", zap.String("data", hex.EncodeToString(accountChange.NewData)))
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
