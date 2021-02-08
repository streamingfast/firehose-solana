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
			if instruction.ProgramId != serum.PROGRAM_ID.String() {
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
				i.SetAccounts(accounts)
			}

			if trxIdx >= len(accountChangesBundle.Transactions) {
				return nil, fmt.Errorf("trx index is out of range, slot: %d (%s), trx index: %d, trx count: %d", slot.Number, slot.Id, trxIdx, len(accountChangesBundle.Transactions))
			}

			trxAccChanges := accountChangesBundle.Transactions[trxIdx]

			if instIdx >= len(trxAccChanges.Instructions) {
				//2
				//4zsYLBJpeyX5VCnzF8rGjkDjYgs39P3qgJd8hwjT9RReBZEKvEALNnWmYdS6tr835Gt2yGLoeamwTUtoyQtFiL36
				//1
				//4

				fmt.Println(instruction.Ordinal)
				fmt.Println(transaction.Id)
				fmt.Println(len(trxAccChanges.Instructions))
				fmt.Println(len(transaction.Instructions))
				fmt.Println(slot.Number)

				return nil, fmt.Errorf("inst index is out of range, slot: %d (%s), trx index: %d, inst index: %d, inst count: %d", slot.Number, slot.Id, trxIdx, instIdx, len(trxAccChanges.Instructions))
			}

			accChanges := trxAccChanges.Instructions[instIdx].Changes
			serumSlot.processInstruction(slot.Number, transaction.Index, uint64(instIdx), slot.Block.Time(), decodedInst, accChanges)
		}
	}
	zlog.Debug("preprocessed slot completed",
		zap.Stringer("slot", blk),
		zap.Int("trading_account_cached_count", len(serumSlot.tradingAccountCache)),
		zap.Int("fill_count", len(serumSlot.fills)),
		zap.Duration("duration", time.Since(t0)),
	)
	return serumSlot, nil
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
