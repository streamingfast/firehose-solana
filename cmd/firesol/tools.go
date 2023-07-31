package main

import (
	"fmt"
	"io"

	"github.com/streamingfast/bstream"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
)

func printBlock(blk *bstream.Block, alsoPrintTransactions bool, out io.Writer) error {
	block := blk.ToProtocol().(*pbsol.Block)

	transactionCount := len(block.Transactions)

	if _, err := fmt.Fprintf(out, "Slot #%d (%s) (prev: %s): %d transactions\n",
		block.GetFirehoseBlockNumber(),
		block.GetFirehoseBlockID(),
		block.GetFirehoseBlockParentID()[0:7],
		transactionCount,
	); err != nil {
		return err
	}

	if alsoPrintTransactions {
		for _, transaction := range block.Transactions {
			status := "✅"
			if transaction.Meta.Err != nil {
				status = "❌"
			}
			transaction.AsBase58String()
			if _, err := fmt.Fprintf(out, "  - Transaction %s %s: %d instructions\n", status, transaction.AsBase58String(), len(transaction.Transaction.Message.Instructions)); err != nil {
				return err
			}
		}
	}

	return nil
}
