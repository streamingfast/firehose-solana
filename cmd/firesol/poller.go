package main

import (
	"github.com/spf13/cobra"
	"github.com/streamingfast/firehose-solana/cmd/firesol/bigtable"
	"github.com/streamingfast/firehose-solana/cmd/firesol/rpc"
	"github.com/streamingfast/logging"

	"go.uber.org/zap"
)

func newPollerCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poller",
		Short: "poll blocks from different sources",
		Args:  cobra.ExactArgs(2),
	}

	cmd.AddCommand(bigtable.NewPollerCmd(logger, tracer))
	cmd.AddCommand(rpc.NewPollerCmd(logger, tracer))
	return cmd
}
