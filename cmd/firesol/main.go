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
	Short: "firesol fetching and tooling",
}

func init() {
	logging.InstantiateLoggers(logging.WithDefaultLevel(zap.InfoLevel))
	rootCmd.AddCommand(newFetchCmd(logger, tracer))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}

func newFetchCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "fetch blocks from different sources",
		Args:  cobra.ExactArgs(2),
	}
	cmd.AddCommand(rpc.NewFetchCmd(logger, tracer))
	return cmd
}
