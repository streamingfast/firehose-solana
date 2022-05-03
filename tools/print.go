package tools

import (
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	pbsol "github.com/streamingfast/sf-solana/types/pb/sf/solana/type/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
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
	printCmd.PersistentFlags().String("store", "gs://dfuseio-global-blocks-us/sol-mainnet/v1", "block store")
}

func printOneBlockE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

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
	filePrefix := fmt.Sprintf("%010d", blockNum)
	fmt.Println(filePrefix)
	err = store.Walk(ctx, filePrefix, "", func(filename string) (err error) {
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

		block, err := readerFactory.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading block: %w", err)
		}

		if err = readBlock(block, false); err != nil {
			return err
		}

	}
	return nil
}

func readBlock(blk *bstream.Block, outputDot bool) error {
	block := blk.ToProtocol().(*pbsol.Block)
	blockId := block.ID()
	blockPreviousId := block.PreviousID()
	hasAccountData := hasAccountData(block)

	if outputDot {
		var virt string
		if block.Number != block.Number {
			virt = " (V)"
		}

		currentID := fmt.Sprintf("%s%s", block.Id[:8], block.Id[len(block.Id)-8:])
		previousID := fmt.Sprintf("%s%s", block.PreviousId[:8], block.PreviousId[len(block.PreviousId)-8:])
		fmt.Printf(
			"  S%s [label=\"%s..%s\\n#%d%s t=%d lib=%d\"];\n  S%s -> S%s;\n",
			currentID,
			blockId[:8],
			blockId[len(blockId)-8:],
			block.Number,
			virt,
			block.TransactionCount,
			blk.LibNum,
			currentID,
			previousID,
		)

	} else {
		fmt.Printf("Slot #%d (%s) (prev: %s...) (blk: %d) (LIB: %d)  (@: %s): %d transactions, has account data : %t\n",
			block.Num(),
			blockId,
			blockPreviousId[0:6],
			block.Number,
			blk.LibNum,
			block.Time(),
			len(block.Transactions),
			hasAccountData,
		)
	}

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

func hasAccountData(block *pbsol.Block) bool {
	for _, t := range block.Transactions {
		for _, inst := range t.Instructions {
			if len(inst.AccountChanges) > 0 {
				return true
			}

		}
	}
	return false
}