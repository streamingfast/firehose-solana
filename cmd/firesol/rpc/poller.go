package rpc

import (
	"fmt"
	"path"
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

	cmd.Flags().Duration("interval-between-fetch", 0, "interval between fetch")

	return cmd
}

func pollerRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		rpcEndpoint := args[0]

		dataDir := sflags.MustGetString(cmd, "data-dir")
		stateDir := path.Join(dataDir, "poller-state")

		firstStreamableBlock, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse first streamable block %d: %w", firstStreamableBlock, err)
		}

		fetchInterval := sflags.MustGetDuration(cmd, "interval-between-fetch")

		logger.Info(
			"launching firehose-solana poller",
			zap.String("rpc_endpoint", rpcEndpoint),
			zap.String("data_dir", dataDir),
			zap.String("state_dir", stateDir),
			zap.Uint64("first_streamable_block", firstStreamableBlock),
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

		requestedBlock, err := rpcClient.GetBlock(ctx, latestSlot)

		err = poller.Run(ctx, firstStreamableBlock, bstream.NewBlockRef(requestedBlock.Blockhash.String(), latestSlot))
		if err != nil {
			return fmt.Errorf("running poller: %w", err)
		}

		return nil
	}
}
