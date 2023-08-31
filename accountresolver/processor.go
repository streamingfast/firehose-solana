package accountsresolver

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/streamingfast/bstream"

	"github.com/mr-tron/base58"
	"github.com/streamingfast/dstore"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/solana-go/programs/addresstablelookup"
)

const AddressTableLookupAccountProgram = "AddressLookupTab1e1111111111111111111111111"

type Cursor struct {
	slotNum   uint64
	blockHash []byte
}

func NewCursor(blockNum uint64, blockHash []byte) *Cursor {
	return &Cursor{
		slotNum:   blockNum,
		blockHash: blockHash,
	}
}

type Processor struct {
	accountsResolver AccountsResolver
	cursor           *Cursor
	readerName       string
}

func NewProcessor(readerName string, cursor *Cursor, accountsResolver AccountsResolver) *Processor {
	return &Processor{
		readerName:       readerName,
		accountsResolver: accountsResolver,
		cursor:           cursor,
	}
}

func (p *Processor) ProcessMergeBlocks(ctx context.Context, store dstore.Store) error {
	startBlockNum := p.cursor.slotNum - p.cursor.slotNum%100
	paddedBlockNum := fmt.Sprintf("%010d", startBlockNum)

	err := store.WalkFrom(ctx, "", paddedBlockNum, func(filename string) error {
		return p.processMergeBlocksFile(ctx, filename, store)
	})

	if err != nil {
		return fmt.Errorf("walking merge block store: %w", err)
	}
	return nil
}

func (p *Processor) processMergeBlocksFile(ctx context.Context, filename string, store dstore.Store) error {
	fmt.Println("Processing merge block file", filename)
	firstBlockOfFile, err := strconv.Atoi(strings.TrimLeft(filename, "0"))
	if err != nil {
		return fmt.Errorf("converting filename to block number: %w", err)
	}
	reader, err := store.OpenObject(ctx, filename)
	if err != nil {
		return fmt.Errorf("opening merge block file %s: %w", filename, err)
	}
	defer reader.Close()

	blockReader, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		return fmt.Errorf("creating block reader for file %s: %w", filename, err)
	}

	for {
		block, err := blockReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading block: %w", err)
		}

		blk := block.ToProtocol().(*pbsol.Block)
		if blk.Slot < uint64(firstBlockOfFile) || blk.Slot <= p.cursor.slotNum {
			fmt.Println("skip block", blk.Slot)
			continue
		}
		err = p.ProcessBlock(context.Background(), blk)
		if err != nil {
			return fmt.Errorf("processing block: %w", err)
		}

		//todo: store new block to new merge file
	}
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
	p.cursor = NewCursor(block.Slot, []byte(block.Blockhash))
	err := p.accountsResolver.StoreCursor(ctx, p.readerName, p.cursor)
	if err != nil {
		return fmt.Errorf("storing cursor at block %d: %w", block.Slot, err)
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
		fmt.Println("Applying address table lookup for :", base58.Encode(addressTableLookup.AccountKey), "count", len(accs))
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
		err := p.ProcessInstruction(ctx, blockNum, base58.Encode(confirmedTransaction.Transaction.Message.AccountKeys[idx]), accountKeys, compiledInstruction)
		if err != nil {
			return fmt.Errorf("confirmedTransaction %s processing compiled instruction: %w", getTransactionHash(confirmedTransaction.Transaction.Signatures), err)
		}
		//todo; only inner instructions of compiled instructions
		if compileIndex+1 > len(confirmedTransaction.Meta.InnerInstructions) {
			continue
		}
		inner := confirmedTransaction.Meta.InnerInstructions[compileIndex]
		for _, instruction := range inner.Instructions {
			err := p.ProcessInstruction(ctx, blockNum, base58.Encode(confirmedTransaction.Transaction.Message.AccountKeys[instruction.ProgramIdIndex]), accountKeys, instruction)
			if err != nil {
				return fmt.Errorf("confirmedTransaction %s processing instruxction: %w", getTransactionHash(confirmedTransaction.Transaction.Signatures), err)
			}
		}
	}

	return nil
}

func (p *Processor) ProcessInstruction(ctx context.Context, blockNum uint64, programAccount string, accountKeys [][]byte, instructionable pbsol.Instructionable) error {
	if programAccount != AddressTableLookupAccountProgram {
		return nil
	}

	instruction := instructionable.ToInstruction()
	if addresstablelookup.ExtendAddressTableLookupInstruction(instruction.Data) {
		tableLookupAccount := accountKeys[instruction.Accounts[0]]
		newAccounts := addresstablelookup.ParseNewAccounts(instruction.Data[12:])
		fmt.Println("Extending address table lookup for:", base58.Encode(tableLookupAccount))
		for _, account := range newAccounts {
			fmt.Println("\t", base58.Encode(account))
		}
		err := p.accountsResolver.Extended(ctx, blockNum, tableLookupAccount, NewAccounts(newAccounts))
		if err != nil {
			return fmt.Errorf("extending address table %s at block %d: %w", tableLookupAccount, blockNum, err)
		}
	}

	return nil
}

func getTransactionHash(signatures [][]byte) string {
	return base58.Encode(signatures[0])
}
