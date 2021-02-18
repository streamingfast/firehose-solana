package serumhist

import (
	"fmt"
	"time"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/serum"
	"go.uber.org/zap"
)

func (i *Injector) preprocessSlot(blk *bstream.Block) (interface{}, error) {
	t0 := time.Now()
	slot := blk.ToNative().(*pbcodec.Slot)

	serumSlot := newSerumSlot()

	var err error
	var accountChangesBundle *pbcodec.AccountChangesBundle

	for trxIdx, transaction := range slot.Transactions {
		if traceEnabled {
			zlog.Debug("processing new transaction",
				zap.String("transaction_id", transaction.Id),
				zap.Int("instruction_count", len(transaction.Instructions)),
			)
		}

		for instIdx, instruction := range transaction.Instructions {
			// FIXME: The DEX v3 address is not known yet, we will need to update this when the address is known
			if instruction.ProgramId != serum.DEXProgramIDV2.String() {
				if traceEnabled {
					zlog.Debug("skipping non-serum instruction",
						zap.Uint64("slot_number", slot.Number),
						zap.String("transaction_id", transaction.Id),
						zap.Int("instruction_index", instIdx),
						zap.String("program_id", instruction.ProgramId),
					)
				}
				continue
			}

			if accountChangesBundle == nil {
				retryCount := 0
				accountChangesBundle, err = slot.Retrieve(i.ctx, func(fileName string) bool {
					retryCount++
					zlog.Debug("account changes file not found...sleeping and retrying",
						zap.Int("retry_count", retryCount),
						zap.String("filename", fileName),
						zap.String("slot_id", slot.Id),
						zap.Uint64("slot_id", slot.Number),
					)
					time.Sleep(time.Duration(retryCount) * 15 * time.Millisecond)
					return true
				})
				if err != nil {
					return nil, fmt.Errorf("unable to retrieve account changes for slot %d (%s): %w", slot.Number, slot.Id, err)
				}
			}

			var decodedInst *serum.Instruction
			if err := bin.NewDecoder(instruction.Data).Decode(&decodedInst); err != nil {
				zlog.Warn("unable to decode serum instruction skipping",
					zap.Uint64("slot_number", slot.Number),
					zap.String("transaction_id", transaction.Id),
					zap.Int("instruction_index", instIdx),
				)
				continue
			}

			if traceEnabled {
				zlog.Debug("processing serum instruction",
					zap.Uint64("slot_number", slot.Number),
					zap.Int("instruction_index", instIdx),
					zap.String("transaction_id", transaction.Id),
					zap.Uint32("serum_instruction_variant_index", decodedInst.TypeID),
				)
			}

			accounts, err := transaction.InstructionAccountMetaList(instruction)
			if err != nil {
				return nil, fmt.Errorf("get instruction account meta list: %w", err)
			}

			if i, ok := decodedInst.Impl.(solana.AccountSettable); ok {
				err = i.SetAccounts(accounts)
				if err != nil {
					return nil, fmt.Errorf("setting account: transaction id: %s: instruction index: %d: %w", transaction.Id, instIdx, err)
				}
			}

			if trxIdx >= len(accountChangesBundle.Transactions) {
				return nil, fmt.Errorf("trx index is out of range, slot: %d (%s), trx index: %d, trx count: %d", slot.Number, slot.Id, trxIdx, len(accountChangesBundle.Transactions))
			}

			trxAccChanges := accountChangesBundle.Transactions[trxIdx]
			if instIdx >= len(trxAccChanges.Instructions) {
				return nil, fmt.Errorf("inst index is out of range, slot: %d (%s), trx index: %d, inst index: %d, inst count: %d", slot.Number, slot.Id, trxIdx, instIdx, len(trxAccChanges.Instructions))
			}

			accChanges := trxAccChanges.Instructions[instIdx].Changes
			err = serumSlot.processInstruction(slot.Number, uint32(transaction.Index), uint32(instIdx), transaction.Id, slot.Id, slot.Block.Time(), decodedInst, accChanges)
			if err != nil {
				return nil, fmt.Errorf("processing instruction: %w", err)
			}
		}
	}
	zlog.Debug("preprocessed slot completed",
		zap.Stringer("slot", blk),
		zap.Int("trading_account_cached_count", len(serumSlot.TradingAccountCache)),
		zap.Int("fill_count", len(serumSlot.OrderFilledEvents)),
		zap.Duration("duration", time.Since(t0)),
	)
	return serumSlot, nil
}

func findAccountChange(accountChanges []*pbcodec.AccountChange, filter func(f *serum.AccountFlag) bool) (*pbcodec.AccountChange, error) {
	for _, accountChange := range accountChanges {
		var f *serum.AccountFlag
		//assumption data should begin with serum prefix "736572756d"
		if err := bin.NewDecoder(accountChange.PrevData[5:]).Decode(&f); err != nil {
			return nil, fmt.Errorf("get account change: unable to decode account flag: %w", err)
		}
		if filter(f) {
			return accountChange, nil
		}
	}
	return nil, nil
}
