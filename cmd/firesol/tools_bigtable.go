package main

import (
	"github.com/spf13/cobra"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func newToolsBigtableCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{Use: "bt", Short: "Solana bigtable sub command"}
	cmd.PersistentFlags().String("bt-project", "mainnet-beta", "Bigtable project")
	cmd.PersistentFlags().String("bt-instance", "solana-ledger", "Bigtable instance")

	cmd.AddCommand(newToolsBigtableBlockCmd(logger, tracer))
	cmd.AddCommand(newToolsBigTableBlocksCmd(logger, tracer))

	return cmd
}
