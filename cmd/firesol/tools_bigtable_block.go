package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"cloud.google.com/go/bigtable"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/bt"
	pbsolv1 "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func newToolsBigtableBlockCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	return &cobra.Command{
		Use:   "block <block_num>",
		Short: "get a block from bigtable",
		Args:  cobra.ExactArgs(1),
		RunE:  bigtableBlockRunE(logger, tracer),
	}
}

func bigtableBlockRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
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

		btClient := bt.New(client, 10, logger, tracer)

		startBlockNum, err := strconv.ParseUint(blockNumStr, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse block number %s: %w", blockNumStr, err)
		}
		endBlockNum := startBlockNum + 1
		fmt.Println("Looking for block: ", startBlockNum)

		foundBlock := false
		if err = btClient.ReadBlocks(ctx, startBlockNum, endBlockNum, false, func(block *pbsolv1.Block) error {
			// the block range may return the next block if it cannot find it
			if block.Slot != startBlockNum {
				return nil
			}

			foundBlock = true
			fmt.Println("Found bigtable row")
			cnt, err := json.MarshalIndent(block, "", " ")
			if err != nil {
				return fmt.Errorf("unable to json marshal block: %w", err)
			}
			fmt.Println(string(cnt))
			return nil
		}); err != nil {
			return fmt.Errorf("failed to find block %d: %w", startBlockNum, err)
		}
		if !foundBlock {
			fmt.Printf("Could not find desired block %d\n", startBlockNum)
		}
		return nil
	}
}
