package tools

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go/programs/serum"
	"github.com/dfuse-io/solana-go/text"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:   "print-block {file} {transaction}",
	Short: "Prints the content summary of a local merged blocks file",
	Args:  cobra.ExactArgs(2),
	RunE:  printBlockE,
}

func init() {
	Cmd.AddCommand(blockCmd)

	blockCmd.Flags().Bool("transactions", false, "Include transaction IDs in output")
}

func printBlockE(cmd *cobra.Command, args []string) error {
	//printTransactions := viper.GetBool("transactions")
	file := args[0]
	abs, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	transactionID := args[1]

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

	blockReader, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		return fmt.Errorf("unable to read one block: %w", err)
	}

	block, err := blockReader.Read()
	if block == nil {
		return err
	}

	slot := block.ToNative().(*pbcodec.Slot)

	for _, tx := range slot.Transactions {
		if tx.Id == transactionID {

			for _, instruction := range tx.Instructions {
				fmt.Println("instruction from programID:", instruction.ProgramId)
				fmt.Println("account change:", len(instruction.AccountChanges))
				if instruction.ProgramId == "EUqojwWA2rd19FZrzeBncJsm38Jm1hEhE3zsmX3bRc2o" {
					var serumInstruction *serum.Instruction
					err := bin.NewDecoder(instruction.Data).Decode(&serumInstruction)
					if err != nil {
						fmt.Printf("❌ Unable to decode serum instruction: %s\n", err)
						return err
					}

					//text.NewEncoder(os.Stdout).Encode(instruction.AccountChanges, nil)
					text.NewEncoder(os.Stdout).Encode(serumInstruction, nil)
				}
				fmt.Println("------")
			}
		}
	}

	return nil
}
