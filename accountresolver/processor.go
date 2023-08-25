package solana_accounts_resolver

import (
	"context"
	"fmt"
	"github.com/mr-tron/base58"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/solana-go/programs/addresstablelookup"
)

const AddressTableLookupAccountProgram = "AddressLookupTab1e1111111111111111111111111"

type Processor struct {
	*KVDBAccountsResolver
}

func NewProcessor(kvdbAccountsResolver *KVDBAccountsResolver) *Processor {
	return &Processor{
		KVDBAccountsResolver: kvdbAccountsResolver,
	}
}

func (p *Processor) ProcessBlock(block *pbsol.Block) error {
	for _, trx := range block.Transactions {
		if trx.Meta.Err != nil {
			continue
		}

		for _, addressTableLookup := range trx.Transaction.Message.AddressTableLookups {
			accs, err := p.KVDBAccountsResolver.Resolve(context.Background(), block.Slot, addressTableLookup.AccountKey)
			if err != nil {
				return fmt.Errorf("resolving address table %s at block %d: %w", base58.Encode(addressTableLookup.AccountKey), block.Slot, err)
			}
			trx.Transaction.Message.AccountKeys = append(trx.Transaction.Message.AccountKeys, accs.ToBytesArray()...)
		}

		err := p.ProcessTransaction(block.Slot, trx.Transaction)
		if err != nil {
			return fmt.Errorf("processing transaction %s at block %d: %w", getTransactionHash(trx.Transaction.Signatures), block.Slot, err)
		}
	}

	return nil
}

func (p *Processor) ProcessTransaction(blockNum uint64, trx *pbsol.Transaction) error {
	accountKeys := trx.Message.AccountKeys
	for _, compiledInstruction := range trx.Message.Instructions {
		idx := compiledInstruction.ProgramIdIndex
		err := p.ProcessInstruction(blockNum, base58.Encode(trx.Message.AccountKeys[idx]), accountKeys, compiledInstruction)
		if err != nil {
			return fmt.Errorf("trx %s processing compiled instruction: %w", getTransactionHash(trx.Signatures), err)
		}
	}

	for _, instruction := range trx.Message.Instructions {
		err := p.ProcessInstruction(blockNum, base58.Encode(trx.Message.AccountKeys[instruction.ProgramIdIndex]), accountKeys, instruction)
		if err != nil {
			return fmt.Errorf("trx %s processing instruxction: %w", getTransactionHash(trx.Signatures), err)
		}
	}

	return nil
}

func (p *Processor) ProcessInstruction(blockNum uint64, programAccount string, accountKeys [][]byte, instructionable pbsol.Instructionable) error {
	if programAccount != AddressTableLookupAccountProgram {
		return nil
	}

	instruction := instructionable.ToInstruction()
	if addresstablelookup.ExtendAddressTableLookupInstruction(instruction.Data) {
		tableLookupAccount := accountKeys[instruction.Accounts[0]]
		newAccounts := addresstablelookup.ParseNewAccounts(instruction.Data[12:])
		err := p.KVDBAccountsResolver.Extended(context.Background(), blockNum, tableLookupAccount, NewAccounts(newAccounts))
		if err != nil {
			return fmt.Errorf("extending address table %s at block %d: %w", tableLookupAccount, blockNum, err)
		}
	}

	return nil
}

func getTransactionHash(signatures [][]byte) string {
	return base58.Encode(signatures[0])
}
