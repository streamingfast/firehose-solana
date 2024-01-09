package main

import (
	"github.com/spf13/cobra"
	firecore "github.com/streamingfast/firehose-core"
	fhCmd "github.com/streamingfast/firehose-core/cmd"
	"github.com/streamingfast/firehose-core/node-manager/mindreader"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	fhCmd.Main(Chain())
}

func Chain() *firecore.Chain[*pbsol.Block] {
	return &firecore.Chain[*pbsol.Block]{
		ShortName:            "sol",
		LongName:             "Solana",
		ExecutableName:       "firesol",
		FullyQualifiedModule: "github.com/streamingfast/firehose-solana",
		Version:              version,

		BlockFactory: func() firecore.Block { return new(pbsol.Block) },

		BlockIndexerFactories: map[string]firecore.BlockIndexerFactory[*pbsol.Block]{},

		BlockTransformerFactories: map[protoreflect.FullName]firecore.BlockTransformerFactory{},
		ConsoleReaderFactory: func(lines chan string, blockEncoder firecore.BlockEncoder, logger *zap.Logger, tracer logging.Tracer) (mindreader.ConsolerReader, error) {
			panic("should not be used!")
		},

		Tools: &firecore.ToolsConfig[*pbsol.Block]{

			RegisterExtraCmd: func(chain *firecore.Chain[*pbsol.Block], toolsCmd *cobra.Command, zlog *zap.Logger, tracer logging.Tracer) error {
				toolsCmd.AddCommand(newPollerCmd(zlog, tracer))
				//toolsCmd.AddCommand(bigtable.NewBigTableCmd(zlog, tracer))
				return nil
			},
		},
	}
}

// Version value, injected via go build `ldflags` at build time, **must** not be removed or inlined
var version = "dev"
