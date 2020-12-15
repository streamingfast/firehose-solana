package cli

import (
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
)

func init() {
	launcher.RegisterCommonFlags = func(cmd *cobra.Command) error {
		// Common stores configuration flags
		cmd.Flags().String("common-blocks-store-url", MergedBlocksStoreURL, "[COMMON] Store URL (with prefix) where to read/write. Used by: relayer, statedb, trxdb-loader, blockmeta, search-indexer, search-live, search-forkresolver, eosws, accounthist")
		cmd.Flags().String("common-oneblock-store-url", OneBlockStoreURL, "[COMMON] Store URL (with prefix) to read/write one-block files. Used by: mindreader, merger")
		cmd.Flags().String("common-blockstream-addr", RelayerServingAddr, "gRPC endpoint to get real-time blocks")

		// Service addresses
		cmd.Flags().String("common-blockmeta-addr", BlockmetaServingAddr, "gRPC endpoint to reach the Blockmeta. Used by: search-indexer, search-router, search-live, eosws, dgraphql")
		return nil
	}
}
