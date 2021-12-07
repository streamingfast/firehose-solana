package serumhist

import (
	"bytes"
	"fmt"
	"time"

	bin "github.com/streamingfast/binary"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/sf-solana/codec"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/serum"
	"go.uber.org/zap"
)

func (i *Injector) preprocessSlot(blk *bstream.Block) (interface{}, error) {
	t0 := time.Now()
	block := blk.ToNative().(*pbcodec.Block)

	serumBlock := newSerumBlock()

	var err error
	var accountChangesBundle *pbcodec.AccountChangesBundle

	for trxIdx, transaction := range block.Transactions {
		if traceEnabled {
			zlog.Debug("processing new transaction",
				codec.ZapBase58("transaction_id", transaction.Id),
				zap.Int("instruction_count", len(transaction.Instructions)),
			)
		}

		if transaction.Failed {
			continue
		}
		for instIdx, instruction := range transaction.Instructions {
			// FIXME: The DEX v3 address is not known yet, we will need to update this when the address is known
			if !bytes.Equal(instruction.ProgramId, serum.DEXProgramIDV2[:]) {
				if traceEnabled {
					zlog.Debug("skipping non-serum instruction",
						zap.Uint64("slot_number", block.Number),
						codec.ZapBase58("transaction_id", transaction.Id),
						zap.Int("instruction_index", instIdx),
						codec.ZapBase58("program_id", instruction.ProgramId),
					)
				}
				continue
			}

			if accountChangesBundle == nil {
				retryCount := 0
				accountChangesBundle, err = block.Retrieve(i.ctx, func(fileName string) bool {
					retryCount++
					zlog.Debug("account changes file not found...sleeping and retrying",
						zap.Int("retry_count", retryCount),
						zap.String("filename", fileName),
						codec.ZapBase58("block", block.Id),
						zap.Uint64("block_num", block.Number),
					)
					time.Sleep(time.Duration(retryCount) * 15 * time.Millisecond)
					return true
				})
				if err != nil {
					return nil, fmt.Errorf("unable to retrieve account changes for slot %d (%s): %w", block.Number, block.Id, err)
				}
			}

			var decodedInst *serum.Instruction
			if err := bin.NewDecoder(instruction.Data).Decode(&decodedInst); err != nil {
				zlog.Warn("unable to decode serum instruction skipping",
					zap.Uint64("slot_number", block.Number),
					codec.ZapBase58("transaction_id", transaction.Id),
					zap.Int("instruction_index", instIdx),
				)
				continue
			}

			if traceEnabled {
				zlog.Debug("processing serum instruction",
					zap.Uint64("slot_number", block.Number),
					zap.Int("instruction_index", instIdx),
					codec.ZapBase58("transaction_id", transaction.Id),
					zap.Uint32("serum_instruction_variant_index", decodedInst.TypeID),
				)
			}

			accounts := transaction.InstructionAccountMetaList(instruction)
			if i, ok := decodedInst.Impl.(solana.AccountSettable); ok {
				err = i.SetAccounts(accounts)
				if err != nil {
					zlog.Warn("error setting account for instruction",
						codec.ZapBase58("transaction_id", transaction.Id),
						zap.Int("insutrction_index", instIdx),
						zap.Error(err),
					)
					continue
				}
			}

			if trxIdx >= len(accountChangesBundle.Transactions) {
				return nil, fmt.Errorf("trx index is out of range, slot: %d (%s), trx index: %d, trx count: %d", block.Number, block.Id, trxIdx, len(accountChangesBundle.Transactions))
			}

			trxAccChanges := accountChangesBundle.Transactions[trxIdx]
			if instIdx >= len(trxAccChanges.Instructions) {
				return nil, fmt.Errorf("inst index is out of range, slot: %d (%s), trx index: %d, inst index: %d, inst count: %d", block.Number, block.Id, trxIdx, instIdx, len(trxAccChanges.Instructions))
			}

			accChanges := trxAccChanges.Instructions[instIdx].Changes
			err = serumBlock.processInstruction(
				block.Number,
				uint32(transaction.Index),
				uint32(instIdx),
				transaction.Id,
				block.Id,
				block.Time(),
				decodedInst,
				accChanges,
			)
			if err != nil {
				return nil, fmt.Errorf("processing instruction: %w", err)
			}
		}
	}
	zlog.Debug("preprocessed slot completed",
		zap.Stringer("slot", blk),
		zap.Int("trading_account_cached_count", len(serumBlock.TradingAccountCache)),
		zap.Int("fill_count", len(serumBlock.OrderFilledEvents)),
		zap.Duration("duration", time.Since(t0)),
	)
	return serumBlock, nil
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
