package bigtable

import (
	"github.com/spf13/cobra"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewBigTableCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{Use: "poller", Short: "poller"}
	cmd.PersistentFlags().String("bt-project", "mainnet-beta", "Bigtable project")
	cmd.PersistentFlags().String("bt-instance", "solana-ledger", "Bigtable instance")

	cmd.AddCommand(NewPollerCmd(logger, tracer))
	return cmd
}
