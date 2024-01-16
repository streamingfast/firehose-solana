package rpc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-core/blockpoller"
	"github.com/streamingfast/firehose-solana/block/fetcher"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewPollerCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rpc <rpc-endpoint> <first-streamable-block>",
		Short: "poll blocks from rpc endpoint",
		Args:  cobra.ExactArgs(2),
		RunE:  pollerRunE(logger, tracer),
	}

	cmd.Flags().String("state-dir", "/data/poller", "interval between fetch")
	cmd.Flags().Duration("interval-between-fetch", 0, "interval between fetch")

	return cmd
}

func pollerRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
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
		)

		rpcClient := rpc.New(rpcEndpoint)

		latestBlockRetryInterval := 250 * time.Millisecond
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
		requestedBlock, err := rpcClient.GetBlockWithOpts(ctx, latestSlot, fetcher.GetBlockOpts)
		if err != nil {
			return fmt.Errorf("getting requested block %d: %w", latestSlot, err)
		}

		err = poller.Run(ctx, startBlock, bstream.NewBlockRef(requestedBlock.Blockhash.String(), latestSlot))
		if err != nil {
			return fmt.Errorf("running poller: %w", err)
		}

		return nil
	}
}
