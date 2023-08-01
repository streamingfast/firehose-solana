package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/streamingfast/cli/sflags"

	"github.com/streamingfast/dlauncher/launcher"
	firecore "github.com/streamingfast/firehose-core"

	"github.com/spf13/cobra"
	"github.com/streamingfast/dstore"
	"go.uber.org/zap"
)

func readerNodeStartBlockResolver(ctx context.Context, command *cobra.Command, runtime *launcher.Runtime, rootLog *zap.Logger) (uint64, error) {
	mergedBlocksStoreURL, _, _, err := firecore.GetCommonStoresURLs(runtime.AbsDataDir)
	if err != nil {
		return 0, err
	}

	mergedBlocksStore, err := dstore.NewDBinStore(mergedBlocksStoreURL)
	if err != nil {
		return 0, fmt.Errorf("unable to create merged blocks store at path %q: %w", mergedBlocksStoreURL, err)
	}

	firstStreamableBlock := sflags.MustGetUint64(command, "common-first-streamable-block")

	return resolveStartBlockNum(ctx, firstStreamableBlock, mergedBlocksStore, rootLog), nil
}

func resolveStartBlockNum(ctx context.Context, start uint64, store dstore.Store, logger *zap.Logger) uint64 {
	errDone := errors.New("done")
	var seenStart *uint64

	err := store.WalkFrom(ctx, "", fmt.Sprintf("%010d", start), func(filename string) error {
		num, err := strconv.ParseUint(filename, 10, 64)
		if err != nil {
			return err
		}

		if num < start { // user has decided to start its merger in the 'future'
			return nil
		}

		if num == start {
			seenStart = &num
			return nil
		}

		// num > start
		if seenStart == nil {
			return errDone // first block after a hole
		}

		// increment by 100
		if num == *seenStart+100 {
			seenStart = &num
			return nil
		}

		return errDone
	})

	if err != nil && !errors.Is(err, errDone) {
		logger.Error("got error walking store", zap.Error(err))
		return start
	}

	switch {
	case seenStart == nil:
		return start // nothing was found
	default:
		return *seenStart + 100
	}

}
