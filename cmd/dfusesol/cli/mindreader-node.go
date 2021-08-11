package cli

import (
	"context"
	"fmt"
	"math"
	"time"

	"google.golang.org/grpc"

	"github.com/streamingfast/bstream/blockstream"
	"github.com/streamingfast/dstore"
	nodeManagerSol "github.com/streamingfast/sf-solana/node-manager"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	nodeManager "github.com/streamingfast/node-manager"
	"github.com/streamingfast/node-manager/mindreader"
	"github.com/streamingfast/sf-solana/codec"
	"go.uber.org/zap"
)

func init() {
	RegisterSolanaNodeApp("mindreader", registerMindreaderNodeFlags)
}

func registerMindreaderNodeFlags(cmd *cobra.Command) error {
	cmd.Flags().String("mindreader-node-grpc-listen-addr", MindreaderNodeGRPCAddr, "Address to listen for incoming gRPC requests")
	cmd.Flags().Bool("mindreader-node-discard-after-stop-num", false, "Ignore remaining blocks being processed after stop num (only useful if we discard the mindreader data after reprocessing a chunk of blocks)")
	cmd.Flags().String("mindreader-node-working-dir", "{dfuse-data-dir}/mindreader/work", "Path where mindreader will stores its files")
	cmd.Flags().String("mindreader-node-block-data-working-dir", "{dfuse-data-dir}/mindreader/block-data-work", "Path where mindreader will stores its files")
	cmd.Flags().Int("mindreader-node-blocks-chan-capacity", 100, "Capacity of the channel holding blocks read by the mindreader. Process will shutdown superviser/geth if the channel gets over 90% of that capacity to prevent horrible consequences. Raise this number when processing tiny blocks very quickly")
	cmd.Flags().Bool("mindreader-node-start-failure-handler", true, "Enables the startup function handler, that gets called if mindreader fails on startup")
	cmd.Flags().Bool("mindreader-node-fail-on-non-contiguous-block", false, "Enables the Continuity Checker that stops (or refuses to start) the superviser if a block was missed. It has a significant performance cost on reprocessing large segments of blocks")
	cmd.Flags().Duration("mindreader-node-wait-upload-complete-on-shutdown", 30*time.Second, "When the mindreader is shutting down, it will wait up to that amount of time for the archiver to finish uploading the blocks before leaving anyway")
	cmd.Flags().Duration("mindreader-node-merge-threshold-block-age", time.Duration(math.MaxInt64), "When processing blocks with a blocktime older than this threshold, they will be automatically merged")
	cmd.Flags().String("mindreader-node-oneblock-suffix", "", "If non-empty, the oneblock files will be appended with that suffix, so that mindreaders can each write their file for a given block instead of competing for writes.")
	cmd.Flags().Bool("mindreader-node-debug-deep-mind", false, "[DEV] Prints deep mind instrumentation logs to standard output, should be use for debugging purposes only")
	cmd.Flags().String("mindreader-node-deepmind-batch-files-path", "/tmp/", "Path where deepmind enabled nodes will write the dmlog batch files, and where the console log will read /tmp/")
	cmd.Flags().Bool("mindreader-node-merge-and-store-directly", false, "[BATCH] When enabled, do not write oneblock files, sidestep the merger and write the merged 100-blocks logs directly to --common-blocks-store-url")
	cmd.Flags().Uint("mindreader-node-start-block-num", 0, "[BATCH] Blocks that were produced with smaller block number then the given block num are skipped")
	cmd.Flags().Uint("mindreader-node-stop-block-num", 0, "[BATCH] Shutdown when we the following 'stop-block-num' has been reached, inclusively.")
	return nil
}

func getMindreaderLogPlugin(
	oneBlockStoreURL string,
	blockDataStoreURL string,
	mergedBlockStoreURL string,
	mergeAndStoreDirectly bool,
	mergeThresholdBlockAge time.Duration,
	workingDir string,
	blockDataWorkingDir string,
	batchStartBlockNum uint64,
	batchStopBlockNum uint64,
	blocksChanCapacity int,
	failOnNonContiguousBlock bool,
	waitTimeForUploadOnShutdown time.Duration,
	oneBlockFileSuffix string,
	operatorShutdownFunc func(error),
	metricsAndReadinessManager *nodeManager.MetricsAndReadinessManager,
	tracker *bstream.Tracker,
	gs *grpc.Server,
	appLogger *zap.Logger,

) (*mindreader.MindReaderPlugin, error) {
	// It's important that this call goes prior running gRPC server since it's doing
	// some service registration. If it's call later on, the overall application exits.
	blockStreamServer := blockstream.NewServer(gs, blockstream.ServerOptionWithLogger(appLogger))

	// blockmetaAddr := viper.GetString("common-blockmeta-addr")
	tracker.AddGetter(bstream.NetworkLIBTarget, func(ctx context.Context) (bstream.BlockRef, error) {
		// FIXME: Need to re-enable the tracker through blockmeta later on (see commented code below), might need to tweak some stuff to make mindreader work...
		return bstream.BlockRefEmpty, nil
	})
	// tracker.AddGetter(bstream.NetworkLIBTarget, bstream.NetworkLIBBlockRefGetter(blockmetaAddr))

	blockDataStore, err := dstore.NewDBinStore(blockDataStoreURL)
	if err != nil {
		return nil, fmt.Errorf("init block data store: %w", err)
	}
	blockDataArchiver := nodeManagerSol.NewBlockDataArchiver(
		blockDataStore,
		blockDataWorkingDir,
		viper.GetString("mindreader-node-block-data-suffix"),
		appLogger,
	)
	if err := blockDataArchiver.Init(); err != nil {
		return nil, fmt.Errorf("init block data archiver: %w", err)
	}
	go blockDataArchiver.Start()

	consoleReaderFactory := func(lines chan string) (mindreader.ConsolerReader, error) {
		return codec.NewConsoleReader(lines, viper.GetString("node-mindreader-deepmind-batch-files-path"))
	}

	consoleReaderTransformer := func(obj interface{}) (*bstream.Block, error) {
		return consoleReaderBlockTransformerWithArchive(blockDataArchiver, obj)
	}

	return mindreader.NewMindReaderPlugin(
		oneBlockStoreURL,
		mergedBlockStoreURL,
		mergeAndStoreDirectly,
		mergeThresholdBlockAge,
		workingDir,
		consoleReaderFactory,
		consoleReaderTransformer,
		tracker,
		batchStartBlockNum,
		batchStopBlockNum,
		blocksChanCapacity,
		metricsAndReadinessManager.UpdateHeadBlock,
		func(error) {
			operatorShutdownFunc(nil)
		},
		failOnNonContiguousBlock,
		waitTimeForUploadOnShutdown,
		oneBlockFileSuffix,
		blockStreamServer,
		appLogger,
	)
}
