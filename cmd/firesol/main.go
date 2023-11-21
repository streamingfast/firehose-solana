package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	pbbstream "github.com/streamingfast/bstream/types/pb/sf/bstream/v1"
	"github.com/streamingfast/cli/sflags"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/dstore"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-core/node-manager/mindreader"
	"github.com/streamingfast/firehose-solana/cmd/firesol/bigtable"
	"github.com/streamingfast/firehose-solana/codec"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func init() {
	firecore.UnsafePayloadKind = pbbstream.Protocol_SOLANA
	firecore.UnsafeResolveReaderNodeStartBlock = readerNodeStartBlockResolver

}

func main() {
	firecore.Main(&firecore.Chain[*pbsol.Block]{
		ShortName:            "sol",
		LongName:             "Solana",
		ExecutableName:       "firesol",
		FullyQualifiedModule: "github.com/streamingfast/firehose-solana",
		Version:              version,

		BlockFactory: func() firecore.Block { return new(pbsol.Block) },

		BlockIndexerFactories: map[string]firecore.BlockIndexerFactory[*pbsol.Block]{},

		BlockTransformerFactories: map[protoreflect.FullName]firecore.BlockTransformerFactory{},

		ConsoleReaderFactory: func(lines chan string, blockEncoder firecore.BlockEncoder, logger *zap.Logger, tracer logging.Tracer) (mindreader.ConsolerReader, error) {
			return codec.NewBigtableConsoleReader(lines, blockEncoder, logger)
		},

		Tools: &firecore.ToolsConfig[*pbsol.Block]{

			RegisterExtraCmd: func(chain *firecore.Chain[*pbsol.Block], toolsCmd *cobra.Command, zlog *zap.Logger, tracer logging.Tracer) error {
				toolsCmd.AddCommand(newPollerCmd(zlog, tracer))
				toolsCmd.AddCommand(bigtable.NewBigTableCmd(zlog, tracer))
				return nil
			},
		},
	})
}

// Version value, injected via go build `ldflags` at build time, **must** not be removed or inlined
var version = "dev"

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
