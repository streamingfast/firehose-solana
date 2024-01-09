package bigtable

import (
	"encoding/base64"
	"fmt"
	"strconv"

	googleBigtable "cloud.google.com/go/bigtable"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/block/fetcher"
	pbsolv1 "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func NewPollerCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bigtable <start_block_num> <stop_block_num>",
		Short: "poll blocks from bigtable",
		Args:  cobra.ExactArgs(2),
		RunE:  pollerRunE(logger, tracer),
	}

	cmd.Flags().String("bt-project", "mainnet-beta", "Bigtable project")
	cmd.Flags().String("bt-instance", "solana-ledger", "Bigtable instance")

	return cmd
}

func pollerRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		startBlockNumStr := args[0]
		stopBlockNumStr := args[1]

		btProject := sflags.MustGetString(cmd, "bt-project")
		btInstance := sflags.MustGetString(cmd, "bt-instance")

		logger.Info("retrieving from bigtable",
			zap.String("start_block_num", startBlockNumStr),
			zap.String("stop_block_num", stopBlockNumStr),
			zap.String("bt_project", btProject),
			zap.String("bt_instance", btInstance),
		)

		client, err := googleBigtable.NewClient(ctx, btProject, btInstance)
		if err != nil {
			return err
		}
		startBlockNum, err := strconv.ParseUint(startBlockNumStr, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse start block number %s: %w", startBlockNumStr, err)
		}

		stopBlockNum, err := strconv.ParseUint(stopBlockNumStr, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse stop block number %s: %w", stopBlockNumStr, err)
		}

		blockReader := fetcher.NewBigtableReader(client, 10, logger, tracer)

		return blockReader.Read(ctx, startBlockNum, stopBlockNum, func(block *pbsolv1.Block) error {
			cnt, err := proto.Marshal(block)
			if err != nil {
				return fmt.Errorf("failed to proto  marshal pb sol block: %w", err)
			}
			b64Cnt := base64.StdEncoding.EncodeToString(cnt)
			lineCnt := fmt.Sprintf("FIRE BLOCK %d %s", block.Slot, b64Cnt)
			if _, err := fmt.Println(lineCnt); err != nil {
				return fmt.Errorf("failed to write log line (char lenght %d): %w", len(lineCnt), err)
			}
			return nil
		})
	}
}
