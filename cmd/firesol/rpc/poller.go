package rpc

import (
	"github.com/spf13/cobra"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewPollerCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rpc <start_block_num> <stop_block_num>",
		Short: "poll blocks from rpc endpoint",
		Args:  cobra.ExactArgs(2),
		RunE:  pollerRunE(logger, tracer),
	}

	cmd.Flags().String("rpc-endpoint", "", "RPC endpoint")

	return cmd
}

func pollerRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) (err error) {
		//ctx := cmd.Context()
		return nil
	}
}
