package cli

import "github.com/spf13/cobra"

func init() {
	RegisterSolanaNodeApp("peering", func(cmd *cobra.Command) error {
		return nil
	})
}
