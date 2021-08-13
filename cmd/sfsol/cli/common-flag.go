package cli

import (
	_ "github.com/streamingfast/dauth/authenticator/null" // register authenticator plugin
	_ "github.com/streamingfast/dauth/ratelimiter/null"   // register ratelimiter plugin

	"github.com/spf13/cobra"
	"github.com/streamingfast/dlauncher/launcher"
)

func init() {
	launcher.RegisterCommonFlags = func(cmd *cobra.Command) error {
		// Network config
		cmd.Flags().String("common-network-id", NetworkID, "[COMMON] Solana network identifier known to us for pre-configured elements Used by: miner-node, mindreader-node")
		cmd.Flags().String("common-sf-network-id", SFNetworkID, "[COMMON] StreamingFast network ID, used for some billing functions by dgraphql")

		// Common stores configuration flags
		cmd.Flags().String("common-blocks-store-url", MergedBlocksStoreURL, "[COMMON] Store URL (with prefix) where to read/write. Used by: relayer, statedb, trxdb-loader, blockmeta, search-indexer, search-live, search-forkresolver, eosws, accounthist")
		cmd.Flags().String("common-oneblock-store-url", OneBlockStoreURL, "[COMMON] Store URL (with prefix) to read/write one-block files. Used by: mindreader, merger")
		cmd.Flags().String("common-block-data-store-url", BlockDataStoreURL, "[COMMON] Store URL (with prefix) to read/write one-block files. Used by: mindreader, merger")
		cmd.Flags().String("common-snapshots-store-url", SnapshotsURL, "Storage bucket with path prefix where state snapshots should be done. Ex: gs://example/snapshots")

		// Service addresses
		cmd.Flags().String("common-blockmeta-addr", BlockmetaServingAddr, "[COMMON]gRPC endpoint to reach the Blockmeta. Used by: search-indexer, search-router, search-live, eosws, dgraphql")
		cmd.Flags().String("common-blockstream-addr", RelayerServingAddr, "[COMMON]gRPC endpoint to get real-time blocks")
		cmd.Flags().String("common-firehose-addr", FirehoseGRPCServingAddr, "[COMMON]gRPC endpoint to get firehose blocks")

		//// Authentication, metering and rate limiter plugins
		cmd.Flags().String("common-auth-plugin", "null://", "[COMMON] Auth plugin URI, see streamingfast/dauth repository")
		cmd.Flags().String("common-metering-plugin", "null://", "[COMMON] Metering plugin URI, see streamingfast/dmetering repository")
		cmd.Flags().String("common-ratelimiter-plugin", "null://", "[COMMON] Rate Limiter plugin URI, see streamingfast/dauth repository")

		//// RPC access
		cmd.Flags().String("common-rpc-endpoint", MinerNodeRPCPort, "[COMMON] RPC endpoint to use to perform Solana JSON-RPC. Used by: dgraphql")
		cmd.Flags().String("common-rpc-ws-endpoint", MinerNodeRPCWSPort, "[COMMON] RPC endpoint to use to perform Solana JSON-RPC. Used by: dgraphql")

		// System Behavior
		cmd.Flags().Duration("common-system-shutdown-signal-delay", 0, "[COMMON] Add a delay between receiving SIGTERM signal and shutting down apps. 'dgraphql', '*-node' will respond negatively to /healthz during this period")

		return nil
	}
}
