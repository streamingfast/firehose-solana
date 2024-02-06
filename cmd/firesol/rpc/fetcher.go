package rpc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-core/blockpoller"
	"github.com/streamingfast/firehose-solana/block/fetcher"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewFetchCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rpc <rpc-endpoint> <first-streamable-block>",
		Short: "fetch blocks from rpc endpoint",
		Args:  cobra.ExactArgs(2),
		RunE:  fetchRunE(logger, tracer),
	}

	cmd.Flags().String("state-dir", "/data/poller", "interval between fetch")
	cmd.Flags().Duration("interval-between-fetch", 0, "interval between fetch")
	cmd.Flags().Duration("latest-block-retry-interval", time.Second, "interval between fetch")
	cmd.Flags().Int("block-fetch-batch-size", 10, "Number of blocks to fetch in a single batch")

	return cmd
}

func fetchRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		rpcEndpoint := args[0]

		stateDir := sflags.MustGetString(cmd, "state-dir")

		startBlock, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse first streamable block %d: %w", startBlock, err)
		}

		fetchInterval := sflags.MustGetDuration(cmd, "interval-between-fetch")

		logger.Info(
			"launching firehose-solana poller",
			zap.String("rpc_endpoint", rpcEndpoint),
			zap.String("state_dir", stateDir),
			zap.Uint64("first_streamable_block", startBlock),
			zap.Duration("interval_between_fetch", fetchInterval),
			zap.Duration("latest_block_retry_interval", sflags.MustGetDuration(cmd, "latest-block-retry-interval")),
		)

		rpcClient := rpc.New(rpcEndpoint)

		latestBlockRetryInterval := sflags.MustGetDuration(cmd, "latest-block-retry-interval")
		poller := blockpoller.New(
			fetcher.NewRPC(rpcClient, fetchInterval, latestBlockRetryInterval, logger),
			blockpoller.NewFireBlockHandler("type.googleapis.com/sf.solana.type.v1.Block"),
			blockpoller.WithStoringState(stateDir),
			blockpoller.WithLogger(logger),
		)

		latestSlot, err := rpcClient.GetSlot(ctx, rpc.CommitmentConfirmed)
		if err != nil {
			return fmt.Errorf("getting latest block: %w", err)
		}

		logger.Info("Found latest slot", zap.Uint64("slot_number", latestSlot))

		err = poller.Run(ctx, startBlock, sflags.MustGetInt(cmd, "block-fetch-batch-size"))
		if err != nil {
			return fmt.Errorf("running poller: %w", err)
		}

		return nil
	}
}
