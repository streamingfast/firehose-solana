package tools

import (
	"github.com/spf13/cobra"
)

var bigtableCmd = &cobra.Command{
	Use:   "bt",
	Short: "Solana bigtable sub command",
}

func init() {
	bigtableCmd.PersistentFlags().String("bt-project", "mainnet-beta", "Bigtable project")
	bigtableCmd.PersistentFlags().String("bt-instance", "solana-ledger", "Bigtable instance")
	Cmd.AddCommand(bigtableCmd)
}
