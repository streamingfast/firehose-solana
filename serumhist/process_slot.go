package serumhist

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	bin "github.com/dfuse-io/binary"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	kvdb "github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/diff"
	"github.com/dfuse-io/solana-go/programs/serum"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

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

			instAccIndexes := instructionAccountIndexes(transaction.AccountKeys, instruction.AccountKeys)
			accounts, err := transaction.AccountMetaList()
			if err != nil {
				return fmt.Errorf("get trx account meta list: %w", err)
			}

			if err = i.processInstruction(ctx, slot.Number, transaction.Id, instruction, serumInstruction, instAccIndexes, accounts); err != nil {
				return fmt.Errorf("process serum instruction: %w", err)
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

func filterAccountChange(accountChanges []*pbcodec.AccountChange, filter func(f *serum.AccountFlag) bool) (*pbcodec.AccountChange, error) {
	zlog.Debug("filtering account change", zap.Int("account_change_count", len(accountChanges)))
	for _, accountChange := range accountChanges {
		var f *serum.AccountFlag
		//assumption data should begin with serum prefix "736572756d"
		if err := bin.NewDecoder(accountChange.PrevData[5:]).Decode(&f); err != nil {
			return nil, fmt.Errorf("get account change: unable to deocde account flag: %w", err)
		}
		zlog.Debug("about to call filtering func")
		if filter(f) {
			return accountChange, nil
		}
	}
	return nil, nil
}

func generateNewOrderKeys(slotNumber uint64, side serum.Side, trader, market solana.PublicKey, old *serum.RequestQueue, new *serum.RequestQueue) (out []*kvdb.KV) {
	zlog.Debug("generate new order kv")
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Requests[#]"); match {
			zlog.Debug("match Requests[#]")
			request := event.Element().Interface().(*serum.Request)
			orderSeqNum := extractOrderSeqNum(side, request.OrderID)
			switch event.Kind {
			case diff.KindChanged:
				zlog.Debug("event KindChanged")
				// this is probably a partial fill we don't care about this right now
			case diff.KindAdded:
				zlog.Debug("event KindAdded")
				// either a cancel request or ad new order
				// this should create keys
				switch request.RequestFlags {

				case serum.RequestFlagNewOrder:
					orderByMarketPubKey := keyer.EncodeOrdersByMarketPubkey(trader, market, orderSeqNum, slotNumber)
					orderByPubKey := keyer.EncodeOrdersByPubkey(trader, market, orderSeqNum, slotNumber)

					zlog.Debug("serum RequestFlagNewOrder",
						zap.Stringer("trader", trader),
						zap.Stringer("market", market),
						zap.Uint64("order_seq_num", orderSeqNum),
						zap.Uint64("slot_num", slotNumber),
						zap.Stringer("order_by_market_pub_key", orderByMarketPubKey),
						zap.Stringer("order_by_pub_key", orderByPubKey),
					)

					out = append(out, &kvdb.KV{
						Key:   orderByMarketPubKey,
						Value: nil,
					})
					out = append(out, &kvdb.KV{
						Key:   orderByPubKey,
						Value: nil,
					})
				case serum.RequestFlagCancelOrder:
					zlog.Debug("serum.RequestFlagCancelOrder")
				}
			case diff.KindRemoved:
				// this is a request that for either canceled or fully filled
				zlog.Debug("diff.KindRemoved")
			default:
			}
		}
	}))
	zlog.Debug("generated new order kv", zap.Int("kv_count", len(out)))
	return out
}

func generateFillKeyValue(slotNumber uint64, market solana.PublicKey, old *serum.EventQueue, new *serum.EventQueue) (out []*kvdb.KV) {
	zlog.Debug("generate fill keys")
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		if match, _ := event.Match("Events[#]"); match {
			e := event.Element().Interface().(*serum.Event)
			switch event.Kind {
			case diff.KindChanged:
				zlog.Debug("diff event change")
				// this is probably a partial fill we don't care about this right now
			case diff.KindAdded:
				zlog.Debug("diff event added", zap.Stringer("flag_string", e.Flag), zap.Uint8("flag", uint8(e.Flag)))
				if e.Flag.IsFill() {
					zlog.Debug("it is a fill")
					size := 16
					buf := make([]byte, size)
					binary.LittleEndian.PutUint64(buf, e.OrderID.Lo)
					binary.LittleEndian.PutUint64(buf[(size/2):], e.OrderID.Hi)
					fill := &pbserumhist.Fill{
						Trader:            e.Owner.String(),
						OrderId:           hex.EncodeToString(buf),
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
				zlog.Debug("diff event remove")
				// this is a request that for either canceled or fully filled
			default:
				zlog.Debug("diff event default")
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
		zlog.Warn("unable to decode 'event queue' old data", zap.String("data", hex.EncodeToString(accountChange.PrevData)))
		return nil, nil, fmt.Errorf("unable to decode 'event queue' old data: %w", err)
	}

	if err := bin.NewDecoder(accountChange.NewData).Decode(&new); err != nil {
		zlog.Warn("unable to decode 'event queue' new data", zap.String("data", hex.EncodeToString(accountChange.NewData)))
		return nil, nil, fmt.Errorf("unable to decode 'event queue' new data: %w", err)
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
