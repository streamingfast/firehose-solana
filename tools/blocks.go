package tools

import (
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var blocksCmd = &cobra.Command{
	Use:   "print-blocks",
	Short: "Prints the content summary of a local merged blocks file",
	Args:  cobra.ExactArgs(1),
	RunE:  printBlocksE,
}

func init() {
	Cmd.AddCommand(blocksCmd)

	blocksCmd.Flags().Bool("transactions", false, "Include transaction IDs in output")
}

func printBlocksE(cmd *cobra.Command, args []string) error {
	printTransactions := viper.GetBool("transactions")
	file := args[0]
	abs, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	dir := path.Dir(abs)
	storeURL := fmt.Sprintf("file://%s", dir)

	compression := ""
	if strings.HasSuffix(file, "zst") || strings.HasSuffix(file, "zstd") {
		compression = "zstd"
	}
	store, err := dstore.NewStore(storeURL, "", compression, false)
	if err != nil {
		return err
	}

	filename := path.Base(abs)
	reader, err := store.OpenObject(context.Background(), filename)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks filename %s: %s\n", filename, err)
		return err
	}
	defer reader.Close()

	readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks filename %s: %s\n", filename, err)
		return err
	}

	seenBlockCount := 0
	for {
		block, err := readerFactory.Read()
		if block != nil {
			seenBlockCount++

			payloadSize := len(block.PayloadBuffer)
			slot := block.ToNative().(*pbcodec.Slot)

			fmt.Printf("Slot #%d (%s) (prev: %s) (%d bytes): %d transactions\n",
				slot.Num(),
				slot.ID(),
				slot.PreviousId,
				payloadSize,
				len(slot.Transactions),
			)
			if printTransactions {
				fmt.Println("- Transactions: ")
				for _, t := range slot.Transactions {
					fmt.Sprintf("  * Trx %s: %d insutrctions/n", t.Id, len(t.Instructions))
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
