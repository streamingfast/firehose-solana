package main

import (
	"cloud.google.com/go/bigtable"
	"fmt"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/bt"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"

	"github.com/streamingfast/cli/sflags"

	"github.com/mr-tron/base58"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/token"
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

func newPrintTransactionCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	transactionCmd := &cobra.Command{
		Use:   "transaction {block_num} {transaction_id}",
		Short: "Prints the content summary of a transaction",
		Long:  "Prints all the content of the transaction given at block num",
		Args:  cobra.ExactArgs(2),
		RunE:  printTransactionE(logger, tracer),
	}
	transactionCmd.PersistentFlags().Bool("decode-token-program", false, "decode token mint to program instruction")
	transactionCmd.PersistentFlags().Bool("bytes-only", false, "print addresses as bytes only")
	return transactionCmd
}

func printTransactionE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		blockNum, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse block number %q: %w", args[0], err)
		}

		transactionId := args[1]

		bytesOnly := sflags.MustGetBool(cmd, "bytes-only")
		_ = bytesOnly

		btProject := sflags.MustGetString(cmd, "bt-project")
		btInstance := sflags.MustGetString(cmd, "bt-instance")

		client, err := bigtable.NewClient(ctx, btProject, btInstance)
		if err != nil {
			return fmt.Errorf("unable to create big table client: %w", err)
		}

		btClient := bt.New(client, 10, logger, tracer)
		foundBlock := false

		if err = btClient.ReadBlocks(ctx, blockNum, blockNum+1, false, func(block *pbsol.Block) error {
			// the block range may return the next block if it cannot find it
			if block.Slot != blockNum {
				return nil
			}

			foundBlock = true
			fmt.Println("Found bigtable row")
			blockOption := &BlockOptions{transactionId: transactionId}
			err := PrintBlock(block, 0, blockOption)
			if err != nil {
				return fmt.Errorf("printing block: %w", err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("failed to find block %d: %w", blockNum, err)
		}
		if !foundBlock {
			fmt.Printf("Could not find desired block %d\n", blockNum)
		}
		return nil
	}
}

type BlockOptions struct {
	transactionId string
	bytesOnly     bool
}

func PrintBlock(block *pbsol.Block, libNum uint64, options *BlockOptions) error {
	blockId := block.Blockhash
	blockPreviousId := block.PreviousBlockhash

	fmt.Printf("Slot #%d (%s) (prev: %s...) (blk: %d) (LIB: %d)  (@: %s)\n",
		block.BlockHeight.BlockHeight,
		blockId,
		blockPreviousId[0:6],
		block.BlockHeight.BlockHeight,
		libNum,
		block.BlockTime,
	)

	for i, trx := range block.Transactions {
		trxId := base58.Encode(trx.Transaction.Signatures[0])

		if options != nil && options.transactionId != trxId {
			continue
		}

		fmt.Printf("Found transaction #%d: %s\n\n", i, trxId)
		fmt.Printf("Header: %s\n\n", trx.Transaction.Message.Header.String())

		fmt.Println("Accounts involved:")
		var accountMetas []*solana.AccountMeta
		for accI, acc := range trx.Transaction.Message.AccountKeys {
			fmt.Printf("\t> Acc [%d]: %s\n", accI, base58.Encode(acc))
			accountMetas = append(accountMetas, solana.NewAccountMeta(solana.PublicKeyFromBytes(acc), false, false))
		}

		fmt.Println("\nRecent BlockHash:", base58.Encode(trx.Transaction.Message.RecentBlockhash))

		fmt.Println("\n Inner Instructions:", trx.Meta.InnerInstructions)

		fmt.Println("Address Account Table Lookup:")
		for _, lookup := range trx.Transaction.Message.AddressTableLookups {
			fmt.Println("\t", lookup.AccountKey)
			fmt.Println("\t\t writable", lookup.WritableIndexes)
			fmt.Println("\t\t read only", lookup.ReadonlyIndexes)
		}

		if len(trx.Transaction.Message.Instructions) > 0 {
			fmt.Println("\nCompiled Instructions:")
		}

		for i, compiledInstruction := range trx.Transaction.Message.Instructions {
			acc := trx.Transaction.Message.AccountKeys[compiledInstruction.ProgramIdIndex]
			if len(acc) == 0 {
				panic(fmt.Sprintf("account isn't part of the transaction accounts, program id index %d", compiledInstruction.ProgramIdIndex))
			}

			programAcc := base58.Encode(acc)

			fmt.Printf("\t> Compiled Instruction [%d]:\n %s\n", i, printCompiledInstructionContent(compiledInstruction, accountMetas))

			if programAcc == token.PROGRAM_ID.String() && viper.GetBool("decode-token-program") {
				fmt.Printf("\nDecoding %s compiled instruction... \n", token.PROGRAM_ID)
				decodedInstruction, err := token.DecodeInstruction(accountMetas, compiledInstruction.Data)
				if err != nil {
					return fmt.Errorf("decoding token instruction: %w", err)
				}

				switch val := decodedInstruction.Impl.(type) {
				case *token.MintTo:
					fmt.Println("\tMintTo - Amount: ", val.Amount)
				case *token.Transfer:
					fmt.Println("\tTransfer Amount: ", val.Amount)
				}
			}
		}

		if trx.Meta.InnerInstructionsNone {
			fmt.Println("No inner instructions")
			continue
		}

		for _, innerInstructions := range trx.Meta.InnerInstructions {
			fmt.Printf("\nInner Instruction [%d]:\n", innerInstructions.Index)
			for j, innerInstruction := range innerInstructions.Instructions {
				acc := trx.Transaction.Message.AccountKeys[innerInstruction.ProgramIdIndex]
				if len(acc) == 0 {
					panic(fmt.Sprintf("account isn't part of the transaction accounts, program id index %d", innerInstruction.ProgramIdIndex))
				}
				fmt.Printf("\t> Instruction [%d] Depth [%d]:\n %s\n", j, innerInstruction.StackHeight, printInnerInstructionContent(innerInstruction, accountMetas))

				programAcc := base58.Encode(acc)
				if programAcc == token.PROGRAM_ID.String() && viper.GetBool("decode-token-program") {
					fmt.Printf("\nDecoding %s inner instruction... \n", token.PROGRAM_ID)
					decodedInstruction, err := token.DecodeInstruction(accountMetas, innerInstruction.Data)
					if err != nil {
						return fmt.Errorf("decoding token innerInstruction: %w", err)
					}

					switch val := decodedInstruction.Impl.(type) {
					case *token.MintTo:
						fmt.Println("\tMintTo - Amount: ", val.Amount)
					case *token.Transfer:
						fmt.Println("\tTransfer Amount: ", val.Amount)
					}
				}
			}
		}
	}

	return nil
}

func printInnerInstructionContent(innerInstruction *pbsol.InnerInstruction, accounts []*solana.AccountMeta) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("\t\tProgram id index: %d\n", innerInstruction.ProgramIdIndex))
	sb.WriteString(fmt.Sprintf("\t\tAccounts:\n"))
	for i, accIdx := range innerInstruction.Accounts {
		if accIdx >= byte(len(accounts)) {
			sb.WriteString(fmt.Sprintf("\t\t\t> Acc [pos: %d, accIdx: %d] not resolved\n", i, accIdx))
			continue
		}
		sb.WriteString(fmt.Sprintf("\t\t\t> Acc [pos: %d, accIdx: %d]: %s\n", i, accIdx, accounts[accIdx].PublicKey.String()))
	}
	sb.WriteString(fmt.Sprintf("\t\tData: %v\n", innerInstruction.Data))
	return sb.String()
}

func printCompiledInstructionContent(compiledInstruction *pbsol.CompiledInstruction, accounts []*solana.AccountMeta) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("\t\tProgram id index: %d\n", compiledInstruction.ProgramIdIndex))
	sb.WriteString(fmt.Sprintf("\t\tAccounts:\n"))
	for i, accIdx := range compiledInstruction.Accounts {
		if accIdx >= byte(len(accounts)) {
			sb.WriteString(fmt.Sprintf("\t\t\t> Acc [pos: %d, accIdx: %d] not resolved\n", i, accIdx))
			continue
		}
		sb.WriteString(fmt.Sprintf("\t\t\t> Acc [pos: %d, accIdx: %d]: %s\n", i, accIdx, accounts[accIdx].PublicKey.String()))
	}
	sb.WriteString(fmt.Sprintf("\t\tData: %v\n", compiledInstruction.Data))
	return sb.String()
}
