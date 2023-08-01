package main

import (
	"encoding/hex"
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/codec"
	"go.uber.org/zap"
)

func newToolsBatchFileReadCmd(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read {batch-file-path}",
		Short: "Reads the content of batch file",
		Args:  cobra.ExactArgs(1),
		RunE:  batchFilesReadRunE(logger),
	}
	cmd.Flags().Bool("detailed", false, "Add instructions logs")
	return cmd
}

func batchFilesReadRunE(logger *zap.Logger) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) error {
		batchFilePath := args[0]
		detailedView := sflags.MustGetBool(cmd, "detailed")

		logger.Info("reading batch file",
			zap.String("path", batchFilePath),
			zap.Bool("detailed", detailedView),
		)

		batch, err := codec.ReadBatchFile(batchFilePath, false, logger)
		if err != nil {
			return fmt.Errorf("unable to read batch file %q: %w", batchFilePath, err)
		}

		fmt.Println("")
		fmt.Printf("Batch %s contains %d transactions\n", batchFilePath, len(batch.Transactions))
		for idx, trx := range batch.Transactions {
			errorIcon := "✅"
			hasError := false
			if trx.Error != nil {
				hasError = true
				errorIcon = "❌"
			}
			fmt.Println("")
			fmt.Printf("%s Trx: %d - %s\n", errorIcon, idx, hex.EncodeToString(trx.Id))
			fmt.Printf("    Indexed: %d\n", trx.Index)
			fmt.Printf("    Failed: %t\n", trx.Failed)
			fmt.Printf("    Inst Count: %d\n", len(trx.Instructions))

			if hasError {
				fmt.Printf("    Error: %s\n", trx.Error.GetError())
			}
			for instIdx, inst := range trx.Instructions {
				programKey := base58.Encode(inst.ProgramId)
				fmt.Printf("    > Inst: %d, Program %s\n", instIdx, programKey)
				fmt.Printf("        Data 0x%s\n", hex.EncodeToString(inst.Data))
				if detailedView {
					fmt.Printf("        Logs:\n")
					for _, log := range inst.Logs {
						fmt.Printf("           * %s\n", log.Message)
					}
				}
			}

		}
		return nil
	}
}
