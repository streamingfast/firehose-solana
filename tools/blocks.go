package tools

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/golang/protobuf/proto"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

var blockDataCmd = &cobra.Command{
	Use:   "block-data {block_num}",
	Short: "Prints the data of a one block file",
	Args:  cobra.ExactArgs(1),
	RunE:  printBlockDataE,
}

var mergedBlocksCmd = &cobra.Command{
	Use:   "blocks {base_block_num}",
	Short: "Prints the content summary of a merged blocks file",
	Args:  cobra.ExactArgs(1),
	RunE:  printMergeBlocksE,
}

func init() {
	Cmd.AddCommand(printCmd)

	printCmd.AddCommand(blockCmd)
	printCmd.AddCommand(mergedBlocksCmd)
	printCmd.AddCommand(blockDataCmd)

	printCmd.PersistentFlags().Uint64("transactions-for-block", 0, "Include transaction IDs in output")
	printCmd.PersistentFlags().Bool("transactions", false, "Include transaction IDs in output")
	printCmd.PersistentFlags().Bool("instructions", false, "Include instruction output")
	printCmd.PersistentFlags().String("store", "gs://dfuseio-global-blocks-us/sol-mainnet/v1", "block store")

	mergedBlocksCmd.Flags().Bool("viz", false, "Output .dot file")
	mergedBlocksCmd.Flags().Bool("data", false, "output block data statistic")

	blockDataCmd.PersistentFlags().String("data-store", "gs://dfuseio-global-blocks-us/sol-mainnet/v1-block-data", "block store")

}

func printMergeBlocksE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	blockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block number %q: %w", args[0], err)
	}

	outputDot := viper.GetBool("viz")
	str := viper.GetString("store")

	store, err := dstore.NewDBinStore(str)
	if err != nil {
		return fmt.Errorf("unable to create store at path %q: %w", store, err)
	}

	filename := fmt.Sprintf("%010d", blockNum)
	reader, err := store.OpenObject(ctx, filename)
	if err != nil {
		fmt.Printf("❌ Unable to read merge blocks filename %s: %s\n", filename, err)
		return err
	}
	defer reader.Close()

	readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks filename %s: %s\n", filename, err)
		return err
	}

	if outputDot {
		fmt.Println("// Run: dot -Tpdf file.dot -o file.pdf")
		fmt.Println("digraph D {")
	} else {
		fmt.Printf("Merged Blocks File: %s\n", store.ObjectURL(filename))
	}
	seenBlockCount := 0
	for {
		err := readBlock(readerFactory, outputDot)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading block: %w", err)
		}
		seenBlockCount++
	}

	if outputDot {
		fmt.Println("}")
	} else {
		fmt.Printf("Total blocks: %d\n", seenBlockCount)
	}

	return nil
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
		err = readBlock(readerFactory, false)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading block: %w", err)
		}

	}
	return nil
}

func printBlockDataE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	blockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block number %q: %w", args[0], err)
	}

	str := viper.GetString("data-store")

	store, err := dstore.NewDBinStore(str)
	if err != nil {
		return fmt.Errorf("unable to create store at path %q: %w", store, err)
	}

	var files []string
	filePrefix := fmt.Sprintf("%010d", blockNum)
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

		bundle := &pbcodec.AccountChangesBundle{}

		data, err := ioutil.ReadAll(reader)
		if err != nil {
			fmt.Printf("❌ Unable to read data from filename %s: %s\n", filepath, err)
			return err
		}

		err = proto.Unmarshal(data, bundle)
		if err != nil {
			fmt.Printf("❌ Unable to unmarshal proto %s: %s\n", filepath, err)
			return err
		}

		totalInst := 0
		for i, transaction := range bundle.Transactions {
			fmt.Printf("* Trx [%d] %s: %d instructions\n", i, transaction.TrxId, len(transaction.Instructions))
			totalInst += len(transaction.Instructions)
			for j, instruction := range transaction.Instructions {
				fmt.Printf("instruction %d changes: %d\n", j, len(instruction.Changes))
			}
		}

		fmt.Println("total inst: ", totalInst)

	}
	return nil
}

func readBlock(reader bstream.BlockReader, outputDot bool) error {
	block, err := reader.Read()
	if err != nil {
		return err
	}

	payloadSize := len(block.PayloadBuffer)

	slot := block.ToNative().(*pbcodec.Slot)
	var accChangesBundle *pbcodec.AccountChangesBundle
	if viper.GetBool("data") {
		store, filename, err := dstore.NewStoreFromURL(slot.AccountChangesFileRef,
			dstore.Compression("zstd"),
		)
		if err != nil {
			return fmt.Errorf("unable to create block data store from url: %s: %w", filename, err)
		}

		reader, err := store.OpenObject(context.Background(), filename)
		if err != nil {
			return fmt.Errorf("unable to open block data: %s : %w", filename, err)
		}
		defer reader.Close()

		data, err := ioutil.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("unable to read all: %s : %w", filename, err)
		}

		accChangesBundle = &pbcodec.AccountChangesBundle{}
		err = proto.Unmarshal(data, accChangesBundle)
		if err != nil {
			return fmt.Errorf("unable to proto unmarshal account changed: %s : %w", filename, err)
		}
	}

	if outputDot {
		var virt string
		if slot.Number != slot.Block.Number {
			virt = " (V)"
		}
		fmt.Printf(
			"  S%s [label=\"%s..%s\\n#%d%s t=%d lib=%d\"];\n  S%s -> S%s;\n",
			slot.Id[:8],
			slot.Id[:8],
			slot.Id[len(slot.Id)-8:],
			slot.Number,
			virt,
			slot.TransactionCount,
			slot.Block.RootNum,
			slot.Id[:8],
			slot.PreviousId[:8],
		)

	} else {
		fmt.Printf("Slot #%d (%s) (prev: %s...) (blk: %d) (LIB: %d) (%d bytes) (@: %s): %d transactions\n",
			slot.Num(),
			slot.ID(),

			slot.PreviousId[0:6],
			slot.Block.Number,
			slot.Block.RootNum,
			payloadSize,
			slot.Block.Time(),
			len(slot.Transactions),
		)
	}

	if viper.GetBool("transactions") || viper.GetUint64("transactions-for-block") == slot.Number {
		totalInstr := 0
		fmt.Println("- Transactions: ")
		for trxIdx, t := range slot.Transactions {
			trxStr := fmt.Sprintf("    * Trx [%d] %s: %d instructions", trxIdx, t.Id, len(t.Instructions))
			if accChangesBundle != nil {
				if trxIdx < len(accChangesBundle.Transactions) {
					trxStr = fmt.Sprintf("%s ✅ acc change", trxStr)
				} else {
					trxStr = fmt.Sprintf("%s ❌ invalid account change index mismatch (%d,%d)", trxStr, trxIdx, len(accChangesBundle.Transactions))
				}
			}
			trxStr = fmt.Sprintf("%s ", trxStr)
			fmt.Println(trxStr)
			totalInstr += len(t.Instructions)
			if viper.GetBool("instructions") {
				for instrx, inst := range t.Instructions {
					instStr := fmt.Sprintf("      * Inst [%d]: program_id %s", inst.Ordinal, inst.ProgramId)
					if accChangesBundle != nil {
						if instrx < len(accChangesBundle.Transactions[trxIdx].Instructions) {
							instStr = fmt.Sprintf("%s ✅ account change", trxStr)
						} else {
							instStr = fmt.Sprintf("%s ❌ invalid account change index mismatch (%d,%d)", trxStr, instrx, len(accChangesBundle.Transactions[trxIdx].Instructions))
						}
					}
					instStr = fmt.Sprintf("%s ", instStr)
					fmt.Println(instStr)
				}
			}

		}
		fmt.Println("total instruction:", totalInstr)
		fmt.Println()
	}
	return nil
}
