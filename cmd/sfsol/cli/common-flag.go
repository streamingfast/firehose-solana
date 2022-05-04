package cli

import (
	"fmt"
	"strings"

	"github.com/streamingfast/cli"
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
		cmd.Flags().Uint64("common-protocol-first-streamable-block", 0, "[COMMON] first chain streamable block. Not genesis")

		cmd.Flags().Bool("common-blocks-cache-enabled", false, FlagDescription(`
				[COMMON] Use a disk cache to store the blocks data to disk and instead of keeping it in RAM. By enabling this, block's Protobuf content, in bytes,
				is kept on file system instead of RAM. This is done as soon the block is downloaded from storage. This is a tradeoff between RAM and Disk, if you
				are going to serve only a handful of concurrent requests, it's suggested to keep is disabled, if you encounter heavy RAM consumption issue, specially
				by the firehose component, it's definitely a good idea to enable it and configure it properly through the other 'common-blocks-cache-...' flags. The cache is
				split in two portions, one keeping N total bytes of blocks of the most recently used blocks and the other one keeping the N earliest blocks as
				requested by the various consumers of the cache.
			`))
		cmd.Flags().String("common-blocks-cache-dir", BlocksCacheDirectory, FlagDescription(`
				[COMMON] Blocks cache directory where all the block's bytes will be cached to disk instead of being kept in RAM.
				This should be a disk that persists across restarts of the Firehose component to reduce the the strain on the disk
				when restarting and streams reconnects. The size of disk must at least big (with a 10% buffer) in bytes as the sum of flags'
				value for  'common-blocks-cache-max-recent-entry-bytes' and 'common-blocks-cache-max-entry-by-age-bytes'.
			`))
		cmd.Flags().Int("common-blocks-cache-max-recent-entry-bytes", 20*1024^3, FlagDescription(`
				[COMMON] Blocks cache max size in bytes of the most recently used blocks, after the limit is reached, blocks are evicted from the cache.
			`))
		cmd.Flags().Int("common-blocks-cache-max-entry-by-age-bytes", 20*1024^3, FlagDescription(`
				[COMMON] Blocks cache max size in bytes of the earliest used blocks, after the limit is reached, blocks are evicted from the cache.
			`))

		// Common stores configuration flags
		cmd.Flags().String("common-blocks-store-url", MergedBlocksStoreURL, "[COMMON] Store URL (with prefix) where to read/write. Used by: relayer, statedb, trxdb-loader, blockmeta, search-indexer, search-live, search-forkresolver, eosws, accounthist")
		cmd.Flags().String("common-oneblock-store-url", OneBlockStoreURL, "[COMMON] Store URL (with prefix) to read/write one-block files. Used by: mindreader, merger")
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

func FlagDescription(in string, args ...interface{}) string {
	return fmt.Sprintf(strings.Join(strings.Split(string(cli.Description(in)), "\n"), " "), args...)
}
