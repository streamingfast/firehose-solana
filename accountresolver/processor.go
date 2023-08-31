package solana_accounts_resolver

import (
	"context"
	"fmt"

	"github.com/mr-tron/base58"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/solana-go/programs/addresstablelookup"
)

const AddressTableLookupAccountProgram = "AddressLookupTab1e1111111111111111111111111"

type Cursor struct {
	blockNum  uint64
	blockHash []byte
}

func newCursor(blockNum uint64, blockHash []byte) *Cursor {
	return &Cursor{
		blockNum:  blockNum,
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

func (p *Processor) ProcessBlock(ctx context.Context, block *pbsol.Block) error {
	if p.cursor == nil {
		return fmt.Errorf("cursor is nil")
	}
	if p.cursor.blockNum != block.Slot {
		return fmt.Errorf("cursor block num %d is not the same as block num %d", p.cursor.blockNum, block.Slot)
	}

	for _, trx := range block.Transactions {
		if trx.Meta.Err != nil {
			continue
		}

		err := p.applyTableLookup(ctx, block.Slot, trx)
		if err != nil {
			return fmt.Errorf("applying table lookup at block %d: %w", block.Slot, err)
		}

		err2 := p.manageAddressLookup(ctx, block.Slot, err, trx)
		if err2 != nil {
			return err2
		}
	}
	p.cursor = newCursor(block.Slot, []byte(block.Blockhash))
	err := p.accountsResolver.StoreCursor(ctx, p.cursor)
	if err != nil {
		return fmt.Errorf("storing cursor at block %d: %w", block.Slot, err)
	}
	return nil
}

func (p *Processor) manageAddressLookup(ctx context.Context, blockNum uint64, err error, trx *pbsol.ConfirmedTransaction) error {
	err = p.ProcessTransaction(ctx, blockNum, trx.Transaction)
	if err != nil {
		return fmt.Errorf("managing address lookup: %w", err)
	}
	return nil
}

func (p *Processor) applyTableLookup(ctx context.Context, blockNum uint64, trx *pbsol.ConfirmedTransaction) error {
	for _, addressTableLookup := range trx.Transaction.Message.AddressTableLookups {
		accs, _, err := p.accountsResolver.Resolve(ctx, blockNum, addressTableLookup.AccountKey)
		if err != nil {
			return fmt.Errorf("resolving address table %s at block %d: %w", base58.Encode(addressTableLookup.AccountKey), blockNum, err)
		}
		trx.Transaction.Message.AccountKeys = append(trx.Transaction.Message.AccountKeys, accs.ToBytesArray()...)
	}
	return nil
}

func (p *Processor) ProcessTransaction(ctx context.Context, blockNum uint64, trx *pbsol.Transaction) error {
	accountKeys := trx.Message.AccountKeys
	for _, compiledInstruction := range trx.Message.Instructions {
		idx := compiledInstruction.ProgramIdIndex
		err := p.ProcessInstruction(ctx, blockNum, base58.Encode(trx.Message.AccountKeys[idx]), accountKeys, compiledInstruction)
		if err != nil {
			return fmt.Errorf("trx %s processing compiled instruction: %w", getTransactionHash(trx.Signatures), err)
		}
	}

	for _, instruction := range trx.Message.Instructions {
		err := p.ProcessInstruction(ctx, blockNum, base58.Encode(trx.Message.AccountKeys[instruction.ProgramIdIndex]), accountKeys, instruction)
		if err != nil {
			return fmt.Errorf("trx %s processing instruxction: %w", getTransactionHash(trx.Signatures), err)
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
