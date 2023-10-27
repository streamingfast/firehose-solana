package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/dstore"
	firecore "github.com/streamingfast/firehose-core"
	"go.uber.org/zap"
)

func readerNodeStartBlockResolver(ctx context.Context, command *cobra.Command, runtime *launcher.Runtime, rootLog *zap.Logger) (uint64, error) {
	startBlockNum, userDefined := sflags.MustGetUint64Provided(command, "reader-node-start-block-num")
	if userDefined {
		return startBlockNum, nil
	}

	mergedBlocksStoreURL, _, _, err := firecore.GetCommonStoresURLs(runtime.AbsDataDir)
	if err != nil {
		return 0, err
	}

	mergedBlocksStore, err := dstore.NewDBinStore(mergedBlocksStoreURL)
	if err != nil {
		return 0, fmt.Errorf("unable to create merged blocks store at path %q: %w", mergedBlocksStoreURL, err)
	}

	firstStreamableBlock := sflags.MustGetUint64(command, "common-first-streamable-block")

	t0 := time.Now()
	rootLog.Info("resolving reader node start block",
		zap.Uint64("first_streamable_block", firstStreamableBlock),
		zap.String("merged_block_store_url", mergedBlocksStoreURL),
	)

	lastMergedBlockNum := firecore.LastMergedBlockNum(ctx, firstStreamableBlock, mergedBlocksStore, rootLog)
	if firstStreamableBlock != lastMergedBlockNum {
		startBlockNum = lastMergedBlockNum + 100
	}

	rootLog.Info("start block resolved",
		zap.Duration("elapsed", time.Since(t0)),
		zap.Uint64("start_block", startBlockNum),
		zap.Uint64("last_merged_block_num", lastMergedBlockNum),
	)
	return startBlockNum, nil
}
