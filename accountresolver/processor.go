package solana_accounts_resolver

import (
	"fmt"
	"github.com/mr-tron/base58"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
)

const AddressTableLookupAccountProgram = "AddressLookupTab1e1111111111111111111111111"

type Processor struct {
	*KVDBAccountsResolver
}

func (p *Processor) ProcessBlock(block *pbsol.Block) error {
	for _, trx := range block.Transactions {
		if trx.Meta.Err != nil {
			continue
		}
		err := p.ProcessTransaction(trx.Transaction)
		if err != nil {
			return fmt.Errorf("processing transaction %s at block %d: %w", getTransactionHash(trx.Transaction.Signatures), block.Slot, err)
		}
	}

	return nil
}

func (p *Processor) ProcessTransaction(trx *pbsol.Transaction) error {
	for _, compiledInstruction := range trx.Message.Instructions {
		idx := compiledInstruction.ProgramIdIndex
		addressAccountLookupAccount := base58.Encode(trx.Message.AccountKeys[idx])
		if addressAccountLookupAccount == AddressTableLookupAccountProgram {
			err := p.ProcessCompiledInstruction(compiledInstruction.Data, compiledInstruction.Accounts[0])
			if err != nil {
				return fmt.Errorf("trx %s processing compiled instruction: %w", getTransactionHash(trx.Signatures), err)
			}
		}
	}

	for _, instruction := range trx.Message.Instructions {
		err := p.ProcessInstruction(instruction)
		if err != nil {
			return fmt.Errorf("trx %s processing instruction: %w", getTransactionHash(trx.Signatures), err)
		}
	}

	return nil
}

func (p *Processor) ProcessInstruction(instructionable pbsol.Instructionable) error {
	instruction := instructionable.ToInstruction()
}

func (p *Processor) ProcessCompiledInstruction(data []byte, tableAddressIdx byte) error {

}

func getTransactionHash(signatures [][]byte) string {
	return base58.Encode(signatures[0])
}
