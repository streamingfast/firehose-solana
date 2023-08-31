package main

import (
	"github.com/spf13/cobra"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/codec"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/node-manager/mindreader"
	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"
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

		Protocol:        "SOL",
		ProtocolVersion: 1,

		BlockFactory: func() firecore.Block { return new(pbsol.Block) },

		BlockIndexerFactories: map[string]firecore.BlockIndexerFactory[*pbsol.Block]{},

		BlockTransformerFactories: map[protoreflect.FullName]firecore.BlockTransformerFactory{},

		ConsoleReaderFactory: func(lines chan string, blockEncoder firecore.BlockEncoder, logger *zap.Logger, tracer logging.Tracer) (mindreader.ConsolerReader, error) {
			return codec.NewBigtableConsoleReader(lines, blockEncoder, logger)
		},

		Tools: &firecore.ToolsConfig[*pbsol.Block]{
			BlockPrinter: printBlock,

			RegisterExtraCmd: func(chain *firecore.Chain[*pbsol.Block], toolsCmd *cobra.Command, zlog *zap.Logger, tracer logging.Tracer) error {
				toolsCmd.AddCommand(newToolsBigtableCmd(zlog, tracer))
				toolsCmd.AddCommand(newToolsBatchFileCmd(zlog))
				toolsCmd.AddCommand(newPrintTransactionCmd(nil))
				toolsCmd.AddCommand(processAddressLookupCmd)
				return nil
			},
		},
	})
}

// Version value, injected via go build `ldflags` at build time, **must** not be removed or inlined
var version = "dev"
