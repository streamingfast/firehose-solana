package main

import (
	"fmt"
	"github.com/mr-tron/base58"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/token"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"

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

func newPrintTransactionCmd(logger *zap.Logger) *cobra.Command {
	transactionCmd.PersistentFlags().String("store", "gs://dfuseio-global-blocks-uscentral/sol-mainnet/v1", "block store")
	transactionCmd.PersistentFlags().Bool("decode-token-program", false, "decode token mint to program instruction")
	return transactionCmd
}

var transactionCmd = &cobra.Command{
	Use:   "transaction {block_num} {transaction_id}",
	Short: "Prints the content summary of a transaction",
	Long:  "Prints all the content of the transaction given at block num",
	Args:  cobra.ExactArgs(2),
	RunE:  printTransactionE,
}

func printTransactionE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	blockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block number %q: %w", args[0], err)
	}

	transactionId := args[1]
	str := ""
	fmt.Println("Using store", str)

	store, err := dstore.NewDBinStore(str)
	if err != nil {
		return fmt.Errorf("unable to create store at path %q: %w", store, err)
	}

	var files []string
	bundleFilename := blockNum - (blockNum % 100)
	filePrefix := fmt.Sprintf("%010d", bundleFilename)
	fmt.Println(filePrefix)
	err = store.Walk(ctx, filePrefix, func(filename string) (err error) {
		files = append(files, filename)
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to find on block files: %w", err)
	}

	fmt.Printf("Found %d oneblock files for block number %d\n", len(files), blockNum)

	for _, filepath := range files {
		reader, err := store.OpenObject(ctx, filepath)
		if err != nil {
			fmt.Printf("❌ Unable to read block filename %s: %s\n", filepath, err)
			return err
		}
		defer reader.Close()

		readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
		if err != nil {
			fmt.Printf("❌ Unable to read blocks filename %s: %s\n", filepath, err)
			return err
		}

		fmt.Printf("One Block File: %s\n", store.ObjectURL(filepath))

		for {
			block, err := readerFactory.Read()
			if err != nil {
				if err == io.EOF {
					return fmt.Errorf("block not found: %q", blockNum)
				}
				return fmt.Errorf("reading block: %w", err)
			}

			if blockNum == block.Num() {
				if err = readTransaction(block, transactionId); err != nil {
					return err
				}
				return nil
			}
		}
	}

	return nil
}

func readTransaction(blk *bstream.Block, transactionId string) error {
	libNum := blk.LibNum
	block := blk.ToProtocol().(*pbsol.Block)
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
		if trxId != transactionId {
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
			break
		}

		for i, innerInstruction := range trx.Meta.InnerInstructions {
			fmt.Printf("\nInner Instruction [%d]:\n", i)
			for j, innerInstruction := range innerInstruction.Instructions {
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
		sb.WriteString(fmt.Sprintf("\t\t\t> Acc [pos: %d, accIdx: %d]: %s\n", i, accIdx, accounts[accIdx].PublicKey.String()))
	}
	sb.WriteString(fmt.Sprintf("\t\tData: %v\n", innerInstruction.Data))
	return sb.String()
}

func printCompiledInstructionContent(innerInstruction *pbsol.CompiledInstruction, accounts []*solana.AccountMeta) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("\t\tProgram id index: %d\n", innerInstruction.ProgramIdIndex))
	sb.WriteString(fmt.Sprintf("\t\tAccounts:\n"))
	for i, accIdx := range innerInstruction.Accounts {
		sb.WriteString(fmt.Sprintf("\t\t\t> Acc [pos: %d, accIdx: %d]: %s\n", i, accIdx, accounts[accIdx].PublicKey.String()))
	}
	sb.WriteString(fmt.Sprintf("\t\tData: %v\n", innerInstruction.Data))
	return sb.String()
}
