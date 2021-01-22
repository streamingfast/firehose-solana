package serumhist

import (
	"context"
	"fmt"

	bin "github.com/dfuse-io/binary"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/serum"
	"go.uber.org/zap"
)

func (i *Injector) processSerumSlot(ctx context.Context, slot *pbcodec.Slot) error {
	for _, transaction := range slot.Transactions {
		if traceEnabled {
			zlog.Debug("processing new transaction",
				zap.String("transaction_id", transaction.Id),
				zap.Int("instruction_count", len(transaction.Instructions)),
			)

		}
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

			if traceEnabled {
				zlog.Debug("processing serum instruction",
					zap.Uint64("slot_number", slot.Number),
					zap.Int("instruction_index", idx),
					zap.String("transaction_id", transaction.Id),
					zap.Uint32("serum_instruction_variant_index", serumInstruction.TypeID),
				)
			}

			accounts, err := transaction.InstructionAccountMetaList(instruction)
			if err != nil {
				return fmt.Errorf("get instruction account meta list: %w", err)
			}

			if i, ok := serumInstruction.Impl.(solana.AccountSettable); ok {
				i.SetAccounts(accounts)
			}

			if err = i.processInstruction(ctx, slot.Number, transaction.Index, uint64(idx), transaction.Id, instruction, serumInstruction); err != nil {
				return fmt.Errorf("process serum instruction: %w", err)
			}
		}
	}
	return nil
}

func filterAccountChange(accountChanges []*pbcodec.AccountChange, filter func(f *serum.AccountFlag) bool) (*pbcodec.AccountChange, error) {
	for _, accountChange := range accountChanges {
		var f *serum.AccountFlag
		//assumption data should begin with serum prefix "736572756d"
		if err := bin.NewDecoder(accountChange.PrevData[5:]).Decode(&f); err != nil {
			return nil, fmt.Errorf("get account change: unable to deocde account flag: %w", err)
		}
		if filter(f) {
			return accountChange, nil
		}
	}
	return nil, nil
}
