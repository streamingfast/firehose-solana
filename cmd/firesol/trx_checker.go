package main

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	firecore "github.com/streamingfast/firehose-core"
	fetcher "github.com/streamingfast/firehose-solana/block/fetcher"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func NewTrxChecker(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "check_trx <start-block> <stop-block> <src-blocks-store> <rpc-endpoint>",
		Args: cobra.ExactArgs(4),
		RunE: checkTrx(logger, tracer),
	}

	return cmd
}

var StopBlockReachError = fmt.Errorf("stop block reach")

func checkTrx(zlog *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		startBlock, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("converting start block block to uint64: %w", err)
		}

		stopBlock, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("converting stop block to uint64: %w", err)
		}

		srcStore, err := dstore.NewDBinStore(args[2])
		if err != nil {
			return fmt.Errorf("unable to create source store: %w", err)
		}

		rpcFetcher := fetcher.NewRPC(
			100*time.Millisecond,
			true,
			zlog,
		)

		endpoint := args[3]
		if err != nil {
			return fmt.Errorf("missing endpoint: %w", err)
		}

		rpcClient := rpc.New(endpoint)

		err = srcStore.WalkFrom(ctx, "", fmt.Sprintf("%010d", startBlock), func(filename string) error {
			var fileReader io.Reader
			fileReader, err = srcStore.OpenObject(ctx, filename)
			if err != nil {
				return fmt.Errorf("creating reader: %w", err)
			}

			var blockReader *bstream.DBinBlockReader
			blockReader, err = bstream.NewDBinBlockReader(fileReader)
			if err != nil {
				return fmt.Errorf("creating block reader: %w", err)
			}

			// the source store is a merged file store
			for {
				block, err := blockReader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("error receiving blocks: %w", err)
				}

				rpcBlock, _, err := rpcFetcher.Fetch(ctx, rpcClient, block.Number)
				if err != nil {
					return fmt.Errorf("fetching block: %w", err)
				}

				solBlock := &pbsol.Block{}
				err = proto.Unmarshal(block.Payload.Value, solBlock)
				if err != nil {
					return fmt.Errorf("unmarshalling stored block: %w", err)
				}

				rpcSolBlock := &pbsol.Block{}
				err = proto.Unmarshal(rpcBlock.Payload.Value, rpcSolBlock)
				if err != nil {
					return fmt.Errorf("unmarshalling rpc block: %w", err)
				}

				if len(rpcSolBlock.Transactions) == len(solBlock.Transactions) {
					zlog.Info("found matching block", zap.Uint64("block", solBlock.Slot), zap.Int("transaction_count", len(rpcSolBlock.Transactions)))
				} else {
					zlog.Error("found diff", zap.Uint64("block", solBlock.Slot), zap.Int("rpc_transaction_count", len(rpcSolBlock.Transactions)), zap.Int("stored_transaction_count", len(solBlock.Transactions)))
				}

				if block.Number == stopBlock {
					return StopBlockReachError
				}
			}

			//next 100 file ...
			return nil
		})

		return nil
	}
}
