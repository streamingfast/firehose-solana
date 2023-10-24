package main

import (
	"cloud.google.com/go/bigtable"
	"fmt"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/bt"
	pbsolv1 "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	_ "github.com/streamingfast/kvdb/store/bigkv"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"strconv"
)

func newFindInvalidBlock(logger *zap.Logger, tracer logging.Tracer, chain *firecore.Chain[*pbsolv1.Block]) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find-invalid-block <start-block> <end-block",
		Short: "",
		RunE:  processFindInvalidBlockE(chain, logger, tracer),
		Args:  cobra.ExactArgs(2),
	}

	cmd.Flags().String("rpc-endpoint", "", "Pass in your RPC endpoint")

	return cmd
}

// test out from: 222330713
// to at least: 222530900

func processFindInvalidBlockE(chain *firecore.Chain[*pbsolv1.Block], logger *zap.Logger, tracer logging.Tracer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		rpcClient := rpc.New(sflags.MustGetString(cmd, "rpc-endpoint"))
		_ = rpcClient

		startBlockNum, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing start block: %w", err)
		}

		endBlockNum, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing start block: %w", err)
		}

		btProject := sflags.MustGetString(cmd, "bt-project")
		btInstance := sflags.MustGetString(cmd, "bt-instance")
		linkable := sflags.MustGetBool(cmd, "linkable")

		logger.Info("retrieving from bigtable",
			zap.Bool("linkable", linkable),
			zap.String("bt_project", btProject),
			zap.String("bt_instance", btInstance),
		)

		client, err := bigtable.NewClient(ctx, btProject, btInstance)
		if err != nil {
			return err
		}
		btClient := bt.New(client, 10, logger, tracer)

		return btClient.ReadBlocks(ctx, startBlockNum, endBlockNum, linkable, func(block *pbsolv1.Block) error {
			missingLogMessagesAndInnerInstructions := 0
			for _, trx := range block.Transactions {
				if trx.Meta.Err != nil {
					continue
				}

				if trx.Meta.LogMessagesNone && trx.Meta.InnerInstructionsNone {
					missingLogMessagesAndInnerInstructions++
				}
			}

			if missingLogMessagesAndInnerInstructions > 0 {
				fmt.Printf("Block: %d number of transactions: %d\n", block.Slot, len(block.Transactions))
				fmt.Printf("\tNumber transactions with missing log messags and inner instructions: %d\n", missingLogMessagesAndInnerInstructions)
			}
			return nil
		})
	}
}
