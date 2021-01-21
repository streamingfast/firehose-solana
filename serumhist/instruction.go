package serumhist

import (
	"context"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	kvdb "github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/serum"
	"go.uber.org/zap"
)

// TODO: this is very repetitive we need to optimze the account setting in solana go
// once that is done we can clean all this up!
func (i *Injector) processInstruction(ctx context.Context, slotNumber uint64, trxID string, inst *pbcodec.Instruction, serumInstruction *serum.Instruction, instAccountIndexes []uint8, trxAccounts []*solana.AccountMeta) error {
	var kvs []*kvdb.KV
	var err error

	// we only care about new order instruction that modify the request queue
	if newOrder, ok := serumInstruction.Impl.(*serum.InstructionNewOrder); ok {
		zlog.Debug("processing new order v1",
			zap.Uint64("slot_number", slotNumber),
			zap.String("trx_id", trxID),
			zap.Uint32("instruction_ordinal", inst.Ordinal),
		)

		if kvs, err = processNewOrderV1(slotNumber, newOrder, instAccountIndexes, trxAccounts, inst.AccountChanges); err != nil {
			return fmt.Errorf("processing new order v1 instructions: %w", err)
		}
	} else if newOrderV2, ok := serumInstruction.Impl.(*serum.InstructionNewOrderV2); ok {
		zlog.Debug("processing new order v2",
			zap.Uint64("slot_number", slotNumber),
			zap.String("trx_id", trxID),
			zap.Uint32("instruction_ordinal", inst.Ordinal),
		)

		if kvs, err = processNewOrderV2(slotNumber, newOrderV2, instAccountIndexes, trxAccounts, inst.AccountChanges); err != nil {
			return fmt.Errorf("processing new order v2 instructions: %w", err)
		}
	} else if mathOrder, ok := serumInstruction.Impl.(*serum.InstructionMatchOrder); ok {

		zlog.Debug("processing match order",
			zap.Uint64("slot_number", slotNumber),
			zap.String("trx_id", trxID),
			zap.Uint32("instruction_ordinal", inst.Ordinal),
		)

		if kvs, err = processMatchOrder(slotNumber, mathOrder, instAccountIndexes, trxAccounts, inst.AccountChanges); err != nil {
			return fmt.Errorf("processing match order instructions: %w", err)
		}
	} else {
		zlog.Debug("unhandled serum instruction",
			zap.Uint64("slot_number", slotNumber),
			zap.String("trx_id", trxID),
			zap.Uint32("instruction_ordinal", inst.Ordinal),
		)
	}

	if len(kvs) == 0 {
		return nil
	}

	zlog.Debug("inserting serumhist keys",
		zap.Int("key_count", len(kvs)),
	)

	for _, kv := range kvs {
		if err := i.kvdb.Put(ctx, kv.Key, kv.Value); err != nil {
			zlog.Warn("failed to write key-value", zap.Error(err))
		}
	}
	return nil
}

func processNewOrderV1(slotNum uint64, inst *serum.InstructionNewOrder, instAccountIndexes []uint8, trxAccounts []*solana.AccountMeta, accountChanges []*pbcodec.AccountChange) (out []*kvdb.KV, err error) {
	if err = inst.SetAccounts(trxAccounts, instAccountIndexes); err != nil {
		return nil, fmt.Errorf("set account metas: %w", err)
	}

	if out, err = kvsForNewOrderRequestQueue(slotNum, inst.Side, inst.Accounts.Owner.PublicKey, inst.Accounts.Market.PublicKey, accountChanges); err != nil {
		return nil, fmt.Errorf("generating serumhist keys: %w", err)
	}

	return out, nil
}

func processNewOrderV2(slotNum uint64, inst *serum.InstructionNewOrderV2, instAccountIndexes []uint8, trxAccounts []*solana.AccountMeta, accountChanges []*pbcodec.AccountChange) (out []*kvdb.KV, err error) {
	if err = inst.SetAccounts(trxAccounts, instAccountIndexes); err != nil {
		return nil, fmt.Errorf("set account metas: %w", err)
	}

	if out, err = kvsForNewOrderRequestQueue(slotNum, inst.Side, inst.Accounts.Owner.PublicKey, inst.Accounts.Market.PublicKey, accountChanges); err != nil {
		return nil, fmt.Errorf("generating serumhist keys: %w", err)
	}

	return out, nil
}

func processMatchOrder(slotNum uint64, inst *serum.InstructionMatchOrder, instAccountIndexes []uint8, trxAccounts []*solana.AccountMeta, accountChanges []*pbcodec.AccountChange) (out []*kvdb.KV, err error) {
	if err := inst.SetAccounts(trxAccounts, instAccountIndexes); err != nil {
		return nil, fmt.Errorf("set account metas: %w", err)
	}

	if out, err = kvsForMatchOrderEventQueue(slotNum, inst, accountChanges); err != nil {
		return nil, fmt.Errorf("generating serumhist keys: %w", err)
	}

	return out, nil
}

func kvsForNewOrderRequestQueue(slotNumber uint64, side serum.Side, trader, market solana.PublicKey, accountChanges []*pbcodec.AccountChange) (out []*kvdb.KV, err error) {
	requestQueueAccountChange, err := filterAccountChange(accountChanges, func(f *serum.AccountFlag) bool {
		return f.Is(serum.AccountFlagInitialized) && f.Is(serum.AccountFlagRequestQueue)
	})
	if err != nil {
		return nil, fmt.Errorf("process new order request queue: get account change: %w", err)
	}

	if requestQueueAccountChange == nil {
		return nil, nil
	}

	old, new, err := decodeRequestQueue(requestQueueAccountChange)
	if err != nil {
		return nil, fmt.Errorf("unable to decode request queue change: %w", err)
	}

	return generateNewOrderKeys(slotNumber, side, trader, market, old, new), nil
}

func kvsForMatchOrderEventQueue(slotNumber uint64, inst *serum.InstructionMatchOrder, accountChanges []*pbcodec.AccountChange) (out []*kvdb.KV, err error) {
	//debugHelper(accountChanges)
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

	return generateFillKeyValue(slotNumber, inst.Accounts.Market.PublicKey, old, new), nil
}
