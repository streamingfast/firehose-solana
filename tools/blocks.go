package tools

import (
	"fmt"
	"io"
	"strconv"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var blocksCmd = &cobra.Command{
	Use:   "print-block {block_num}",
	Short: "Prints the content summary of a local one block file",
	Args:  cobra.ExactArgs(1),
	RunE:  printBlocksE,
}

func init() {
	Cmd.AddCommand(blocksCmd)

	blocksCmd.Flags().Bool("transactions", false, "Include transaction IDs in output")
	blocksCmd.Flags().Bool("instructions", false, "Include instruction output")
	blocksCmd.Flags().String("store", "gs://dfuseio-global-blocks-us/sol-mainnet/v1-oneblock", "One block store")
}

func printBlocksE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	printTransactions := viper.GetBool("transactions")
	printInstructions := viper.GetBool("instructions")
	blockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block number %q: %w", args[0], blockNum)
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

		seenBlockCount := 0
		for {
			block, err := readerFactory.Read()
			if block != nil {
				seenBlockCount++

				payloadSize := len(block.PayloadBuffer)
				slot := block.ToNative().(*pbcodec.Slot)
				fmt.Printf("One Block File: %s:\n", filepath)
				fmt.Printf("  Slot #%d (%s) (prev: %s) (%d bytes): %d transactions\n",
					slot.Num(),
					slot.ID(),
					slot.PreviousId,
					payloadSize,
					len(slot.Transactions),
				)
				fmt.Printf("  Block #%d (%s) @ %s\n",
					slot.Block.Number,
					slot.Block.Id,
					slot.Block.Time(),
				)

				if printTransactions {
					fmt.Println("- Transactions: ")
					for _, t := range slot.Transactions {
						fmt.Printf("    * Trx %s: %d instructions\n", t.Id, len(t.Instructions))
						if printInstructions {
							for _, inst := range t.Instructions {
								fmt.Printf("      * Inst [%d]: program_id %s\n", inst.Ordinal, inst.ProgramId)
							}
						}

					}
					fmt.Println()
				}

				continue
			}

			if block == nil && err == io.EOF {
				fmt.Printf("Total blocks: %d\n", seenBlockCount)
				return nil
			}

			if err != nil {
				return err
			}
		}
	}
	return nil
}
