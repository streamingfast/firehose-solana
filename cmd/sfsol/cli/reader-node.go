package cli

import (
	"context"
	"fmt"
	"github.com/streamingfast/firehose-solana/codec"
	"time"

	"github.com/streamingfast/logging"

	"github.com/streamingfast/dlauncher/launcher"

	"github.com/streamingfast/solana-go"

	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/blockstream"
	nodeManager "github.com/streamingfast/node-manager"
	"github.com/streamingfast/node-manager/mindreader"
	"go.uber.org/zap"
)

func init() {

	appLogger, appTracer := logging.PackageLogger("reader", fmt.Sprintf("github.com/streamingfast/firehose-solana/reader"))
	nodeLogger, _ := logging.PackageLogger("node", fmt.Sprintf("github.com/streamingfast/firehose-solana/reader/node"))

	launcher.RegisterApp(zlog, &launcher.AppDef{
		ID:          "reader-node",
		Title:       fmt.Sprintf("Solana Reader"),
		Description: fmt.Sprintf("Solana %s node with built-in operational manager", "reader"),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("reader-node-network", "development", "Which network this node refers to, 'development' ")
			cmd.Flags().String("reader-node-config-dir", "./reader", "Directory for config files")
			cmd.Flags().String("reader-node-data-dir", fmt.Sprintf("{sf-data-dir}/%s/data", "reader"), "Directory for data (node blocks and state)")
			cmd.Flags().String("reader-node-rpc-port", rpcPortByKind["reader"], "HTTP listening port of Solana node, setting this to empty string disable RPC endpoint for the node")
			cmd.Flags().String("reader-node-gossip-port", gossipPortByKind["reader"], "TCP gossip listening port of Solana node")
			cmd.Flags().String("reader-node-p2p-port-start", p2pPortStartByKind["reader"], "P2P dynamic range start listening port of Solana node")
			cmd.Flags().String("reader-node-p2p-port-end", p2pPortEndByKind["reader"], "P2P dynamic range end of Solana node")
			cmd.Flags().String("reader-node-http-listen-addr", httpListenAddrByKind["reader"], "Solana node manager HTTP address when operational command can be send to control the node")
			cmd.Flags().Duration("reader-node-readiness-max-latency", 30*time.Second, "The health endpoint '/healthz' will return an error until the head block time is within that duration to now")
			cmd.Flags().Duration("reader-node-shutdown-delay", 0, "Delay before shutting manager when sigterm received")
			cmd.Flags().String("reader-node-extra-arguments", "", "Extra arguments to be passed when executing superviser binary")
			cmd.Flags().String("reader-node-bootstrap-data-url", "", "URL where to find bootstrapping data for this node, the URL must point to a `.tar.zst` archive containing the full data directory to bootstrap from")
			cmd.Flags().Bool("reader-node-log-to-zap", true, "Enable all node logs to transit into app's logger directly, when false, prints node logs directly to stdout")
			cmd.Flags().Bool("reader-node-rpc-enable-debug-apis", false, "[DEV] Enable some of the Solana validator RPC APIs that can be used for debugging purposes")
			cmd.Flags().Duration("reader-node-startup-delay", 0, "[DEV] wait time before launching")
			cmd.Flags().String("reader-node-restore-snapshot-name", "", "If non-empty, the node will be restored from that snapshot when it starts.")

			cmd.Flags().Duration("reader-node-auto-snapshot-period", 0, "If non-zero, the node manager will check on disk at this period interval to see if the underlying node has produced a snapshot. Use in conjunction with --snapshot-interval-slots in the --reader-node-extra-arguments. Specify 1m, 2m...")
			cmd.Flags().String("reader-node-local-snapshot-folder", "", "where solana snapshots are stored by the node")
			cmd.Flags().Int("reader-node-number-of-snapshots-to-keep", 0, "if non-zero, after a successful snapshot, older snapshots will be deleted to only keep that number of recent snapshots")
			cmd.Flags().String("reader-node-genesis-url", "", "url to genesis.tar.bz2")
			cmd.Flags().String("reader-node-grpc-listen-addr", ReaderNodeGRPCAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("reader-node-working-dir", "{sf-data-dir}/reader/work", "Path where reader will stores its files")
			cmd.Flags().String("reader-node-block-data-working-dir", "{sf-data-dir}/reader/block-data-work", "Path where reader will stores its files")
			cmd.Flags().Int("reader-node-blocks-chan-capacity", 100, "Capacity of the channel holding blocks read by the reader. Process will shutdown superviser/geth if the channel gets over 90% of that capacity to prevent horrible consequences. Raise this number when processing tiny blocks very quickly")
			cmd.Flags().Bool("reader-node-start-failure-handler", true, "Enables the startup function handler, that gets called if reader fails on startup")
			cmd.Flags().Bool("reader-node-fail-on-non-contiguous-block", false, "Enables the Continuity Checker that stops (or refuses to start) the superviser if a block was missed. It has a significant performance cost on reprocessing large segments of blocks")
			cmd.Flags().String("reader-node-merge-threshold-block-age", "24h", "When processing blocks with a blocktime older than this threshold, they will be automatically merged (you can also use \"always\" or \"never\")")
			cmd.Flags().String("reader-node-oneblock-suffix", "", "If non-empty, the oneblock files will be appended with that suffix, so that readers can each write their file for a given block instead of competing for writes.")
			cmd.Flags().Bool("reader-node-debug-firehose-logs", false, "[DEV] Prints firehose logs to standard output, should be use for debugging purposes only")
			cmd.Flags().Bool("reader-node-merge-and-store-directly", false, "[BATCH] When enabled, do not write oneblock files, sidestep the merger and write the merged 100-blocks logs directly to --common-blocks-store-url")
			cmd.Flags().Uint("reader-node-start-block-num", 0, "[BATCH] Blocks that were produced with smaller block number then the given block num are skipped")
			cmd.Flags().Uint("reader-node-stop-block-num", 0, "[BATCH] Shutdown when we the following 'stop-block-num' has been reached, inclusively.")
			cmd.Flags().Bool("reader-node-purge-account-data", false, "When flag enabled, the reader will purge the account changes from the blocks before storing it")
			cmd.Flags().String("reader-node-firehose-batch-files-path", "", "Path where firehose enabled nodes will write the firelog batch files, and where the console log will read /tmp/")
			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) error {
			return nil
		},
		FactoryFunc: nodeFactoryFunc("reader-node", "reader", appLogger, appTracer, nodeLogger),
	})

}

func getConsoleReaderFactory(appLogger *zap.Logger, batchFilePath string, purgeAccountChanges bool) mindreader.ConsolerReaderFactory {
	return func(lines chan string) (mindreader.ConsolerReader, error) {
		zlog.Debug("setting up console reader",
			zap.String("batch_file_path", batchFilePath),
			zap.Bool("purge_account", purgeAccountChanges),
		)
		opts := []codec.ConsoleReaderOption{codec.IgnoreAccountChangesForProgramID(solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111111"))}
		if purgeAccountChanges {
			opts = append(opts, codec.IgnoreAllAccountChanges())
		}
		if batchFilePath != "" {
			opts = append(opts, codec.WithBatchFilesPath(batchFilePath))

		}
		r, err := codec.NewConsoleReader(appLogger, lines, opts...)
		if err != nil {
			return nil, fmt.Errorf("initiating console reader: %w", err)
		}
		return r, nil
	}
}

func getReaderLogPlugin(blockStreamServer *blockstream.Server, oneBlockStoreURL string, mergedBlockStoreURL string, mergeThresholdBlockAge string, workingDir string, blockDataWorkingDir string, batchStartBlockNum uint64, batchStopBlockNum uint64, blocksChanCapacity int, waitTimeForUploadOnShutdown time.Duration, oneBlockFileSuffix string, operatorShutdownFunc func(error), metricsAndReadinessManager *nodeManager.MetricsAndReadinessManager, tracker *bstream.Tracker, appLogger *zap.Logger, appTracer logging.Tracer, batchFilePath string, purgeAccountChanges bool) (*mindreader.MindReaderPlugin, error) {
	tracker.AddGetter(bstream.NetworkLIBTarget, func(ctx context.Context) (bstream.BlockRef, error) {
		// FIXME: Need to re-enable the tracker through blockmeta later on (see commented code below), might need to tweak some stuff to make reader work...
		return bstream.BlockRefEmpty, nil
	})

	consoleReaderFactory := getConsoleReaderFactory(appLogger, batchFilePath, purgeAccountChanges)
	return mindreader.NewMindReaderPlugin(
		oneBlockStoreURL,
		mergedBlockStoreURL,
		mergeThresholdBlockAge,
		workingDir,
		consoleReaderFactory,
		batchStartBlockNum,
		batchStopBlockNum,
		blocksChanCapacity,
		metricsAndReadinessManager.UpdateHeadBlock,
		func(error) {
			operatorShutdownFunc(nil)
		},
		waitTimeForUploadOnShutdown,
		oneBlockFileSuffix,
		blockStreamServer,
		appLogger,
		appTracer,
	)
}
