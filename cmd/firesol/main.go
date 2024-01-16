package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/streamingfast/firehose-solana/cmd/firesol/rpc"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var logger, tracer = logging.PackageLogger("firesol", "github.com/streamingfast/firehose-solana")
var rootCmd = &cobra.Command{
	Use:   "firesol",
	Short: "firesol poller and tooling",
}

func init() {
	logging.InstantiateLoggers(logging.WithDefaultLevel(zap.InfoLevel))
	rootCmd.AddCommand(newPollerCmd(logger, tracer))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}

func newPollerCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poller",
		Short: "poll blocks from different sources",
		Args:  cobra.ExactArgs(2),
	}
	cmd.AddCommand(rpc.NewPollerCmd(logger, tracer))
	return cmd
}
