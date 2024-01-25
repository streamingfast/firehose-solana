package block

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"github.com/streamingfast/bstream/stream"
	"github.com/streamingfast/dstore"
	firecore "github.com/streamingfast/firehose-core"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewBlockCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block <source> <destination> <range>",
		Short: "upgrade-merged-blocks from legacy to new format using anypb.Any as payload",
	}

	cmd.AddCommand(NewFetchCmd(logger, tracer))

	return cmd
}

func NewFetchCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade-merged-blocks <source> <destination> <range>",
		Short: "upgrade-merged-blocks from legacy to new format using anypb.Any as payload",
		Args:  cobra.ExactArgs(4),
		RunE:  getMergedBlockUpgrader(logger),
	}

	return cmd
}

func getMergedBlockUpgrader(rootLog *zap.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		source := args[0]
		sourceStore, err := dstore.NewDBinStore(source)
		if err != nil {
			return fmt.Errorf("reading source store: %w", err)
		}

		dest := args[1]
		destStore, err := dstore.NewStore(dest, "dbin.zst", "zstd", true)
		if err != nil {
			return fmt.Errorf("reading destination store: %w", err)
		}

		start, err := strconv.ParseUint(args[2], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing start block num: %w", err)
		}
		stop, err := strconv.ParseUint(args[3], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing stop block num: %w", err)
		}

		rootLog.Info("starting block upgrader process", zap.Uint64("start", start), zap.Uint64("stop", stop), zap.String("source", source), zap.String("dest", dest))
		writer := &firecore.MergedBlocksWriter{
			Cmd:          cmd,
			Store:        destStore,
			LowBlockNum:  firecore.LowBoundary(start),
			StopBlockNum: stop,
			TweakBlock:   setParentBlockNumber,
		}
		blockStream := stream.New(nil, sourceStore, nil, int64(start), writer, stream.WithFinalBlocksOnly())

		err = blockStream.Run(context.Background())
		if errors.Is(err, io.EOF) {
			rootLog.Info("Complete!")
			return nil
		}
		return err
	}
}

func setParentBlockNumber(block *pbbstream.Block) (*pbbstream.Block, error) {
	b := &pbsol.Block{}
	err := block.Payload.UnmarshalTo(b)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling solana block %d: %w", block.Number, err)
	}

	block.ParentNum = b.ParentSlot
	return block, nil
}
