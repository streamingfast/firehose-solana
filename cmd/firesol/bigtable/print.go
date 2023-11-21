package bigtable

import (
	"fmt"
	"os"
	"strconv"

	"cloud.google.com/go/bigtable"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/mr-tron/base58"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/blockreader"
	pbsolv1 "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func newPrintCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "print <block_num>",
		Short: "print a block from bigtable",
		Args:  cobra.ExactArgs(1),
	}
	cmd.AddCommand(newPrintBlocksCmd(logger, tracer))
	return cmd
}

func newPrintBlocksCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "block",
		RunE:  printBlockRunE(logger, tracer),
	}
	return cmd
}

func printBlockRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		blockNumStr := args[0]
		btProject := sflags.MustGetString(cmd, "bt-project")
		btInstance := sflags.MustGetString(cmd, "bt-instance")

		logger.Info("retrieving from bigtable",
			zap.String("block_num", blockNumStr),
			zap.String("bt_project", btProject),
			zap.String("bt_instance", btInstance),
		)

		client, err := bigtable.NewClient(ctx, btProject, btInstance)
		if err != nil {
			return fmt.Errorf("unable to create big table client: %w", err)
		}

		blockReader := blockreader.NewBigtableReader(client, 10, logger, tracer)

		startBlockNum, err := strconv.ParseUint(blockNumStr, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse block number %s: %w", blockNumStr, err)
		}
		endBlockNum := startBlockNum + 1
		fmt.Println("Looking for block: ", startBlockNum)

		foundBlock := false
		err = blockReader.Read(ctx, startBlockNum, endBlockNum, func(block *pbsolv1.Block) error {
			// the block range may return the next block if it cannot find it
			if block.Slot != startBlockNum {
				return nil
			}

			foundBlock = true
			encoder := jsontext.NewEncoder(os.Stdout)

			var marshallers = json.NewMarshalers(
				json.MarshalFuncV2(func(encoder *jsontext.Encoder, t []byte, options json.Options) error {
					return encoder.WriteToken(jsontext.String(base58.Encode(t)))
				}),
			)

			err := json.MarshalEncode(encoder, block, json.WithMarshalers(marshallers))
			if err != nil {
				return fmt.Errorf("encoding block: %w", err)
			}
			return nil

		})

		if err != nil {
			return fmt.Errorf("failed to find block %d: %w", startBlockNum, err)
		}
		if !foundBlock {
			fmt.Printf("Could not find desired block %d\n", startBlockNum)
		}
		return nil
	}
}
