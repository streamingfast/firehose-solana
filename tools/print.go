package tools

import (
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	"github.com/mr-tron/base58"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	pbsolv1 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v1"
	pbsolv2 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v2"
)

var printCmd = &cobra.Command{
	Use:   "print",
	Short: "Prints of one block or merged blocks file",
}

var blockCmd = &cobra.Command{
	Use:   "block {block_num}",
	Short: "Prints the content summary of a one block file",
	Args:  cobra.ExactArgs(1),
	RunE:  printOneBlockE,
}

func init() {
	Cmd.AddCommand(printCmd)
	printCmd.AddCommand(blockCmd)

	printCmd.PersistentFlags().Uint64("transactions-for-block", 0, "Include transaction IDs in output")
	printCmd.PersistentFlags().Bool("transactions", false, "Include transaction IDs in output")
	printCmd.PersistentFlags().Bool("instructions", false, "Include instruction output")
	printCmd.PersistentFlags().String("store", "gs://dfuseio-global-blocks-uscentral/sol-mainnet/v1", "block store")
}

func printOneBlockE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	augmentedStack := viper.GetBool("global-augmented-mode")
	blockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block number %q: %w", args[0], err)
	}

	str := viper.GetString("store")
	fmt.Println(str)

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
				if err = readBlock(block, augmentedStack); err != nil {
					return err
				}
				return nil
			}
		}
	}
	return nil
}

func readBlock(blk *bstream.Block, augmentedData bool) error {
	if augmentedData {
		return readPBSolBlock(blk.ToProtocol().(*pbsolv2.Block), blk.LibNum)
	}
	return readPBSolanaBlock(blk.ToProtocol().(*pbsolv1.Block), blk.LibNum)
}

func readPBSolBlock(block *pbsolv2.Block, LibNum uint64) error {
	blockId := block.ID()
	blockPreviousId := block.PreviousID()
	hasAccountData := hasAccountData(block)

	fmt.Printf("Slot #%d (%s) (prev: %s...) (blk: %d) (LIB: %d)  (@: %s): %d transactions, has account data : %t\n",
		block.Num(),
		blockId,
		blockPreviousId[0:6],
		block.Number,
		LibNum,
		block.Time(),
		len(block.Transactions),
		hasAccountData,
	)

	if viper.GetBool("transactions") || viper.GetUint64("transactions-for-block") == block.Number {
		totalInstr := 0
		fmt.Println("- Transactions: ")

		for trxIdx, t := range block.Transactions {
			trxStr := fmt.Sprintf("    * ")
			if t.Failed {
				trxStr = fmt.Sprintf("%s ❌", trxStr)
			} else {
				trxStr = fmt.Sprintf("%s ✅", trxStr)
			}

			fmt.Println(fmt.Sprintf("%s Trx [%d] %s: %d instructions ", trxStr, trxIdx, t.Id, len(t.Instructions)))
			accs, _ := t.AccountMetaList()
			for _, acc := range accs {
				fmt.Println("account: ", acc)
			}
			totalInstr += len(t.Instructions)
			if viper.GetBool("instructions") {
				for _, inst := range t.Instructions {
					instStr := fmt.Sprintf("      * Inst [%d]: program_id %s", inst.Index, inst.ProgramId)
					instStr = fmt.Sprintf("%s ", instStr)
					fmt.Println(instStr)
					fmt.Println(hex.EncodeToString(inst.Data))
				}
			}

		}
		fmt.Println("total instruction:", totalInstr)
		fmt.Println()
	}
	return nil
}

func readPBSolanaBlock(block *pbsolv1.Block, LibNum uint64) error {
	blockId := block.ID()
	blockPreviousId := block.PreviousID()

	fmt.Printf("Slot #%d (%s) (prev: %s...) (blk: %d) (LIB: %d)  (@: %s): %d transactions\n",
		block.Num(),
		blockId,
		blockPreviousId[0:6],
		block.Num(),
		LibNum,
		block.Time(),
		len(block.Transactions),
	)
	if viper.GetBool("transactions") || viper.GetUint64("transactions-for-block") == block.Num() {
		totalInstr := 0
		fmt.Println("- Transactions: ")

		for trxIdx, t := range block.Transactions {
			tid := base58.Encode(t.Transaction.Signatures[0])
			trxStr := fmt.Sprintf("    * ")
			if t.Meta.Err != nil {
				trxStr = fmt.Sprintf("%s ❌", trxStr)
			} else {
				trxStr = fmt.Sprintf("%s ✅", trxStr)
			}

			fmt.Println(fmt.Sprintf("%s Trx [%d] %s: %d instructions ", trxStr, trxIdx, tid, len(t.Transaction.Message.Instructions)))
			for idx, acc := range t.Transaction.Message.AccountKeys {
				fmt.Printf("           > Acc [%d]: %s\n", idx, base58.Encode(acc))
			}
			totalInstr += len(t.Transaction.Message.Instructions)
			if viper.GetBool("instructions") {
				for _, inst := range t.Transaction.Message.Instructions {
					instStr := fmt.Sprintf("      * Inst: program_id %d", inst.ProgramIdIndex)
					instStr = fmt.Sprintf("%s ", instStr)
					fmt.Println(instStr)
					fmt.Println(hex.EncodeToString(inst.Data))
				}
			}

		}
		fmt.Println("total instruction:", totalInstr)
		fmt.Println()
	}
	return nil
}
func hasAccountData(block *pbsolv2.Block) bool {
	for _, t := range block.Transactions {
		for _, inst := range t.Instructions {
			if len(inst.AccountChanges) > 0 {
				return true
			}

		}
	}
	return false
}
