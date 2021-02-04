package tools

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

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

	printCmd.PersistentFlags().Bool("transactions", false, "Include transaction IDs in output")
	printCmd.PersistentFlags().Bool("instructions", false, "Include instruction output")
	printCmd.PersistentFlags().String("store", "gs://dfuseio-global-blocks-us/sol-mainnet/v2", "block store")
}

func printMergeBlocksE(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("Merged Blocks File: %s\n", store.ObjectURL(filename))
	seenBlockCount := 0
	for {
		err := readBlock(readerFactory)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Total blocks: %d\n", seenBlockCount)
				return nil
			}
			return fmt.Errorf("reading block: %w", err)
		}
		seenBlockCount++
	}

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
		err = readBlock(readerFactory)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading block: %w", err)
		}

	}
	return nil
}

func readBlock(reader bstream.BlockReader) error {
	block, err := reader.Read()
	if block != nil {
		payloadSize := len(block.PayloadBuffer)

		if block.Number == 63943243 {
			ioutil.WriteFile("/tmp/cochon.log", block.Payload(), 0644)
		}
		slot := block.ToNative().(*pbcodec.Slot)
		fmt.Printf("Slot #%d (%s) (prev: %s...) (%d bytes) (blk: %d) (@: %s): %d transactions\n",
			slot.Num(),
			slot.ID(),
			slot.PreviousId[0:6],
			payloadSize,
			slot.Block.Number,
			slot.Block.Time(),
			len(slot.Transactions),
		)

		if viper.GetBool("transactions") {
			fmt.Println("- Transactions: ")
			for _, t := range slot.Transactions {
				fmt.Printf("    * Trx %s: %d instructions\n", t.Id, len(t.Instructions))
				if viper.GetBool("instructions") {
					for _, inst := range t.Instructions {
						fmt.Printf("      * Inst [%d]: program_id %s\n", inst.Ordinal, inst.ProgramId)
					}
				}

			}
			fmt.Println()
		}

		return nil
	}

	if block == nil && err == io.EOF {
		return io.EOF
	}

	if err != nil {
		return err
	}

	return nil
}
