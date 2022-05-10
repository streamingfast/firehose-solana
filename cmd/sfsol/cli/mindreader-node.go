package cli

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/streamingfast/sf-solana/node-manager/codec"

	"github.com/streamingfast/solana-go"

	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/blockstream"
	nodeManager "github.com/streamingfast/node-manager"
	"github.com/streamingfast/node-manager/mindreader"
	"go.uber.org/zap"
)

func init() {
	RegisterMindreaderSolanaNodeApp(registerMindreaderNodeFlags)
}

func registerMindreaderNodeFlags(cmd *cobra.Command) error {
	cmd.Flags().String("mindreader-node-grpc-listen-addr", MindreaderNodeGRPCAddr, "Address to listen for incoming gRPC requests")
	cmd.Flags().Bool("mindreader-node-discard-after-stop-num", false, "Ignore remaining blocks being processed after stop num (only useful if we discard the mindreader data after reprocessing a chunk of blocks)")
	cmd.Flags().String("mindreader-node-working-dir", "{sf-data-dir}/mindreader/work", "Path where mindreader will stores its files")
	cmd.Flags().String("mindreader-node-block-data-working-dir", "{sf-data-dir}/mindreader/block-data-work", "Path where mindreader will stores its files")
	cmd.Flags().Int("mindreader-node-blocks-chan-capacity", 100, "Capacity of the channel holding blocks read by the mindreader. Process will shutdown superviser/geth if the channel gets over 90% of that capacity to prevent horrible consequences. Raise this number when processing tiny blocks very quickly")
	cmd.Flags().Bool("mindreader-node-start-failure-handler", true, "Enables the startup function handler, that gets called if mindreader fails on startup")
	cmd.Flags().Bool("mindreader-node-fail-on-non-contiguous-block", false, "Enables the Continuity Checker that stops (or refuses to start) the superviser if a block was missed. It has a significant performance cost on reprocessing large segments of blocks")
	cmd.Flags().Duration("mindreader-node-wait-upload-complete-on-shutdown", 30*time.Second, "When the mindreader is shutting down, it will wait up to that amount of time for the archiver to finish uploading the blocks before leaving anyway")
	cmd.Flags().Duration("mindreader-node-merge-threshold-block-age", time.Duration(math.MaxInt64), "When processing blocks with a blocktime older than this threshold, they will be automatically merged")
	cmd.Flags().String("mindreader-node-oneblock-suffix", "", "If non-empty, the oneblock files will be appended with that suffix, so that mindreaders can each write their file for a given block instead of competing for writes.")
	cmd.Flags().Bool("mindreader-node-debug-deep-mind", false, "[DEV] Prints deep mind instrumentation logs to standard output, should be use for debugging purposes only")
	cmd.Flags().Bool("mindreader-node-merge-and-store-directly", false, "[BATCH] When enabled, do not write oneblock files, sidestep the merger and write the merged 100-blocks logs directly to --common-blocks-store-url")
	cmd.Flags().Uint("mindreader-node-start-block-num", 0, "[BATCH] Blocks that were produced with smaller block number then the given block num are skipped")
	cmd.Flags().Uint("mindreader-node-stop-block-num", 0, "[BATCH] Shutdown when we the following 'stop-block-num' has been reached, inclusively.")
	cmd.Flags().Bool("mindreader-node-purge-account-data", false, "When flag enabled, the mindreader will purge the account changes from the blocks before storing it")
	cmd.Flags().String("mindreader-node-deepmind-batch-files-path", "", "Path where deepmind enabled nodes will write the dmlog batch files, and where the console log will read /tmp/")
	return nil
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

func getMindreaderLogPlugin(blockStreamServer *blockstream.Server, oneBlockStoreURL string, mergedBlockStoreURL string, mergeAndStoreDirectly bool, mergeThresholdBlockAge time.Duration, workingDir string, blockDataWorkingDir string, batchStartBlockNum uint64, batchStopBlockNum uint64, blocksChanCapacity int, failOnNonContiguousBlock bool, waitTimeForUploadOnShutdown time.Duration, oneBlockFileSuffix string, operatorShutdownFunc func(error), metricsAndReadinessManager *nodeManager.MetricsAndReadinessManager, tracker *bstream.Tracker, appLogger *zap.Logger, batchFilePath string, purgeAccountChanges bool) (*mindreader.MindReaderPlugin, error) {
	tracker.AddGetter(bstream.NetworkLIBTarget, func(ctx context.Context) (bstream.BlockRef, error) {
		// FIXME: Need to re-enable the tracker through blockmeta later on (see commented code below), might need to tweak some stuff to make mindreader work...
		return bstream.BlockRefEmpty, nil
	})

	consoleReaderFactory := getConsoleReaderFactory(appLogger, batchFilePath, purgeAccountChanges)
	return mindreader.NewMindReaderPlugin(
		oneBlockStoreURL,
		mergedBlockStoreURL,
		mergeAndStoreDirectly,
		mergeThresholdBlockAge,
		workingDir,
		consoleReaderFactory,
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
