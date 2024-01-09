package rpc

import (
	"fmt"
	"path"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
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

		//todo: init fetcher
		//todo: init poller handler
		//todo: init poller

		//todo: fetch latest block from chain rpc

		//todo: run the poller

		return nil
	}
}
