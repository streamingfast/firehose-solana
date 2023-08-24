package solana_accounts_resolver

import (
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
)

type Processor struct {
}

func (p *Processor) ProcessBlock(block *pbsol.Block) error {
	return nil
}

func (p *Processor) ProcessTransaction(block *pbsol.Transaction) error {
	return nil
}
