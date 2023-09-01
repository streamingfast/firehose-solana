package accountsresolver

import (
	"bytes"
	"context"
	"fmt"
	"github.com/mr-tron/base58"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/solana-go/programs/addresstablelookup"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
)

var AddressTableLookupAccountProgram = mustFromBase58("AddressLookupTab1e1111111111111111111111111")
var SystemProgram = mustFromBase58("11111111111111111111111111111111")

type Cursor struct {
	slotNum uint64
}

func NewCursor(blockNum uint64) *Cursor {
	return &Cursor{
		slotNum: blockNum,
	}
}

type Processor struct {
	accountsResolver AccountsResolver
	cursor           *Cursor
	readerName       string
	logger           *zap.Logger
}

func NewProcessor(readerName string, cursor *Cursor, accountsResolver AccountsResolver, logger *zap.Logger) *Processor {
	return &Processor{
		readerName:       readerName,
		accountsResolver: accountsResolver,
		cursor:           cursor,
		logger:           logger,
	}
}

func (p *Processor) ProcessMergeBlocks(ctx context.Context, sourceStore dstore.Store, destinationStore dstore.Store) error {
	startBlockNum := p.cursor.slotNum - p.cursor.slotNum%100
	paddedBlockNum := fmt.Sprintf("%010d", startBlockNum)

	p.logger.Info("Processing merge blocks", zap.Uint64("cursor_block_num", p.cursor.slotNum), zap.String("first_merge_filename", paddedBlockNum))

	err := sourceStore.WalkFrom(ctx, "", paddedBlockNum, func(filename string) error {
		p.logger.Debug("processing merge block file", zap.String("filename", filename))
		return p.processMergeBlocksFile(ctx, filename, sourceStore, destinationStore)
	})

	if err != nil {
		return fmt.Errorf("walking merge block sourceStore: %w", err)
	}

	p.logger.Info("Done processing merge blocks")

	return nil
}

func (p *Processor) processMergeBlocksFile(ctx context.Context, filename string, sourceStore dstore.Store, destinationStore dstore.Store) error {
	p.logger.Info("Processing merge block file", zap.String("filename", filename))
	firstBlockOfFile, err := strconv.Atoi(strings.TrimLeft(filename, "0"))
	if err != nil {
		return fmt.Errorf("converting filename to block number: %w", err)
	}

	reader, err := sourceStore.OpenObject(ctx, filename)
	if err != nil {
		return fmt.Errorf("opening merge block file %s: %w", filename, err)
	}
	defer reader.Close()

	blockReader, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		return fmt.Errorf("creating block reader for file %s: %w", filename, err)
	}

	bundleReader := NewBundleReader(ctx, p.logger)

	go func() {
		for {
			block, err := blockReader.Read()
			if err != nil {
				if err == io.EOF {
					bundleReader.PushError(io.EOF)
					return
				}
				bundleReader.PushError(fmt.Errorf("reading block: %w", err))
				return
			}

			blk := block.ToProtocol().(*pbsol.Block)
			if blk.Slot < uint64(firstBlockOfFile) || blk.Slot <= p.cursor.slotNum {
				p.logger.Debug("skip block", zap.Uint64("slot", blk.Slot))
				continue
			}
			err = p.ProcessBlock(context.Background(), blk)
			if err != nil {
				bundleReader.PushError(fmt.Errorf("processing block: %w", err))
				return
			}
			err = bundleReader.PushBlock(block)
			if err != nil {
				bundleReader.PushError(fmt.Errorf("pushing block to bundle reader: %w", err))
				return
			}
		}
	}()

	err = destinationStore.WriteObject(ctx, filename, bundleReader)
	if err != nil {
		return fmt.Errorf("writing bundle file: %w", err)
	}
	p.logger.Info("new merge blocks file written:", zap.String("filename", filename))

	return nil
}

func (p *Processor) ProcessBlock(ctx context.Context, block *pbsol.Block) error {
	if p.cursor == nil {
		return fmt.Errorf("cursor is nil")
	}

	if p.cursor.slotNum != block.ParentSlot {
		return fmt.Errorf("cursor block num %d is not the same as parent slot num %d of block %d", p.cursor.slotNum, block.ParentSlot, block.Slot)
	}

	for _, trx := range block.Transactions {
		if trx.Meta.Err != nil {
			continue
		}

		err := p.applyTableLookup(ctx, block.Slot, trx)
		if err != nil {
			return fmt.Errorf("applying table lookup at block %d: %w", block.Slot, err)
		}

		err = p.manageAddressLookup(ctx, block.Slot, err, trx)
		if err != nil {
			return fmt.Errorf("managing address lookup at block %d: %w", block.Slot, err)
		}
	}
	return nil
}

func (p *Processor) manageAddressLookup(ctx context.Context, blockNum uint64, err error, trx *pbsol.ConfirmedTransaction) error {
	err = p.ProcessTransaction(ctx, blockNum, trx)
	if err != nil {
		return fmt.Errorf("processing transactions: %w", err)
	}
	return nil
}

func (p *Processor) applyTableLookup(ctx context.Context, blockNum uint64, trx *pbsol.ConfirmedTransaction) error {
	for _, addressTableLookup := range trx.Transaction.Message.AddressTableLookups {
		accs, _, err := p.accountsResolver.Resolve(ctx, blockNum, addressTableLookup.AccountKey)
		p.logger.Info("resolve address table lookup", zap.String("account", base58.Encode(addressTableLookup.AccountKey)), zap.Int("count", len(accs)))
		if err != nil {
			return fmt.Errorf("resolving address table %s at block %d: %w", base58.Encode(addressTableLookup.AccountKey), blockNum, err)
		}
		//todo: should fail if accs is nil
		trx.Transaction.Message.AccountKeys = append(trx.Transaction.Message.AccountKeys, accs.ToBytesArray()...)
	}
	return nil
}

func (p *Processor) ProcessTransaction(ctx context.Context, blockNum uint64, confirmedTransaction *pbsol.ConfirmedTransaction) error {
	accountKeys := confirmedTransaction.Transaction.Message.AccountKeys
	for compileIndex, compiledInstruction := range confirmedTransaction.Transaction.Message.Instructions {
		idx := compiledInstruction.ProgramIdIndex
		err := p.ProcessInstruction(ctx, blockNum, confirmedTransaction.Transaction.Message.AccountKeys[idx], accountKeys, compiledInstruction)
		if err != nil {
			return fmt.Errorf("confirmedTransaction %s processing compiled instruction: %w", getTransactionHash(confirmedTransaction.Transaction.Signatures), err)
		}
		//todo; only inner instructions of compiled instructions
		if compileIndex+1 > len(confirmedTransaction.Meta.InnerInstructions) {
			continue
		}
		inner := confirmedTransaction.Meta.InnerInstructions[compileIndex]
		for _, instruction := range inner.Instructions {
			err := p.ProcessInstruction(ctx, blockNum, confirmedTransaction.Transaction.Message.AccountKeys[instruction.ProgramIdIndex], accountKeys, instruction)
			if err != nil {
				return fmt.Errorf("confirmedTransaction %s processing instruxction: %w", getTransactionHash(confirmedTransaction.Transaction.Signatures), err)
			}
		}
	}

	return nil
}

func (p *Processor) ProcessInstruction(ctx context.Context, blockNum uint64, programAccount Account, accountKeys [][]byte, instructionable pbsol.Instructionable) error {
	if !bytes.Equal(programAccount, AddressTableLookupAccountProgram) {
		return nil
	}

	instruction := instructionable.ToInstruction()
	if addresstablelookup.ExtendAddressTableLookupInstruction(instruction.Data) {
		tableLookupAccount := accountKeys[instruction.Accounts[0]]
		newAccounts := addresstablelookup.ParseNewAccounts(instruction.Data[12:])
		p.logger.Info("Extending address table lookup", zap.String("account", base58.Encode(tableLookupAccount)), zap.Int("new_account_count", len(newAccounts)))
		for _, account := range newAccounts {
			p.logger.Debug("\t new account", zap.String("account", base58.Encode(account)))
		}
		err := p.accountsResolver.Extended(ctx, blockNum, tableLookupAccount, NewAccounts(newAccounts))
		if err != nil {
			return fmt.Errorf("extending address table %s at block %d: %w", tableLookupAccount, blockNum, err)
		}
		p.cursor = NewCursor(blockNum)
		err = p.accountsResolver.StoreCursor(ctx, p.readerName, p.cursor)
		if err != nil {
			return fmt.Errorf("storing cursor at block %d: %w", blockNum, err)
		}
	}

	return nil
}

func getTransactionHash(signatures [][]byte) string {
	return base58.Encode(signatures[0])
}
