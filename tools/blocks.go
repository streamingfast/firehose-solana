package tools

import (
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
	printCmd.AddCommand(blockDataCmd)
	printCmd.AddCommand(blockCmd)
	printCmd.AddCommand(mergedBlocksCmd)

	printCmd.PersistentFlags().Uint64("transactions-for-block", 0, "Include transaction IDs in output")
	printCmd.PersistentFlags().Bool("transactions", false, "Include transaction IDs in output")
	printCmd.PersistentFlags().Bool("instructions", false, "Include instruction output")
	printCmd.PersistentFlags().String("store", "gs://dfuseio-global-blocks-us/sol-mainnet/v1", "block store")
	blockDataCmd.PersistentFlags().String("data-store", "gs://dfuseio-global-blocks-us/sol-mainnet/v1-block-data", "block store")
	mergedBlocksCmd.Flags().Bool("viz", false, "Output .dot file")
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

	// if block.Number == 63943243 {
	// 	ioutil.WriteFile("/tmp/cochon.log", block.Payload(), 0644)
	// }
	slot := block.ToNative().(*pbcodec.Slot)
	if outputDot {
		var virt string
		if slot.Number != slot.Block.Number {
			virt = " (V)"
		}
		fmt.Printf(
			"  S%s [label=\"%s..%s\\n#%d%s t=%d\"];\n  S%s -> S%s;\n",
			slot.Id[:8],
			slot.Id[:8],
			slot.Id[len(slot.Id)-8:],
			slot.Number,
			virt,
			slot.TransactionCount,
			slot.Id[:8],
			slot.PreviousId[:8],
		)

	} else {
		fmt.Printf("Slot #%d (%s) (prev: %s...) (%d bytes) (blk: %d) (@: %s): %d transactions\n",
			slot.Num(),
			slot.ID(),
			slot.PreviousId[0:6],
			payloadSize,
			slot.Block.Number,
			slot.Block.Time(),
			len(slot.Transactions),
		)
	}

	if viper.GetBool("transactions") || viper.GetUint64("transactions-for-block") == slot.Number {
		fmt.Println("- Transactions: ")
		totalInstr := 0
		for i, t := range slot.Transactions {
			fmt.Printf("    * Trx [%d] %s: %d instructions\n", i, t.Id, len(t.Instructions))
			totalInstr += len(t.Instructions)
			if viper.GetBool("instructions") {
				for _, inst := range t.Instructions {
					fmt.Printf("      * Inst [%d]: program_id %s\n", inst.Ordinal, inst.ProgramId)
				}
			}

		}
		fmt.Println("total instruction:", totalInstr)
		fmt.Println()
	}
	return nil
}
