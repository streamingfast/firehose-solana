package main

import (
	"fmt"
	"os"

	"github.com/streamingfast/firehose-solana/cmd/firesol/block"

	"github.com/spf13/cobra"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-core/cmd/tools"
	"github.com/streamingfast/firehose-core/cmd/tools/compare"
	"github.com/streamingfast/firehose-solana/cmd/firesol/rpc"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
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
	chain := &firecore.Chain[*pbsol.Block]{
		BlockFactory: func() firecore.Block { return new(pbsol.Block) },
	}

	rootCmd.AddCommand(tools.ToolsCmd)
	rootCmd.AddCommand(block.NewBlockCmd(logger, tracer))
	tools.ToolsCmd.AddCommand(compare.NewToolsCompareBlocksCmd(chain))
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
