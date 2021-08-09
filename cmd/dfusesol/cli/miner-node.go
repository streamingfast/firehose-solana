package cli

import "github.com/spf13/cobra"

func init() {
	RegisterSolanaNodeApp("miner", func(cmd *cobra.Command) error {
		return nil
	})
}
