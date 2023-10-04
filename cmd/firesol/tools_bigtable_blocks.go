package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	accountsresolver "github.com/streamingfast/firehose-solana/accountresolver"
	kvstore "github.com/streamingfast/kvdb/store"

	"cloud.google.com/go/bigtable"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/bt"
	pbsolv1 "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func newToolsBigTableBlocksCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocks <start_block_num> <stop_block_num>",
		Short: "get a range of blocks from bigtable",
		Args:  cobra.ExactArgs(2),
		RunE:  bigtableBlocksRunE(logger, tracer),
	}

	cmd.Flags().Bool("firehose-enabled", false, "When enable the blocks read will output Firehose formatted logs 'FIRE <block_num> <block_payload_in_hex>'")
	cmd.Flags().Bool("compact", false, "When printing in JSON it will print compact instead of pretty-printed output")
	cmd.Flags().Bool("linkable", false, "Ensure that no block is skipped they are linkeable")
	cmd.Flags().String("table-lookup-dsn", "", "DSN to the table lookup kv db")
	return cmd
}

func bigtableBlocksRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		startBlockNumStr := args[0]
		stopBlockNumStr := args[1]

		firehoseEnabled := sflags.MustGetBool(cmd, "firehose-enabled")
		compact := sflags.MustGetBool(cmd, "compact")
		linkable := sflags.MustGetBool(cmd, "linkable")
		btProject := sflags.MustGetString(cmd, "bt-project")
		btInstance := sflags.MustGetString(cmd, "bt-instance")
		tableLookupDSN := sflags.MustGetString(cmd, "table-lookup-dsn")

		logger.Info("retrieving from bigtable",
			zap.Bool("firehose_enabled", firehoseEnabled),
			zap.Bool("compact", compact),
			zap.Bool("linkable", linkable),
			zap.String("start_block_num", startBlockNumStr),
			zap.String("stop_block_num", stopBlockNumStr),
			zap.String("bt_project", btProject),
			zap.String("bt_instance", btInstance),
			zap.String("table_lookup_dsn", tableLookupDSN),
		)
		client, err := bigtable.NewClient(ctx, btProject, btInstance)
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

		btClient := bt.New(client, 10, logger, tracer)

		db, err := kvstore.New(tableLookupDSN)
		if err != nil {
			return fmt.Errorf("unable to create kv store: %w", err)
		}

		resolver := accountsresolver.NewKVDBAccountsResolver(db, logger)
		processor := accountsresolver.NewProcessor("reader", resolver, logger)

		return btClient.ReadBlocks(ctx, startBlockNum, stopBlockNum, linkable, func(block *pbsolv1.Block) error {
			if firehoseEnabled {
				stats := &accountsresolver.Stats{}
				err := processor.ProcessBlock(ctx, stats, block)
				if err != nil {
					return fmt.Errorf("unable to process table lookup for block: %w", err)
				}
				stats.Log(logger)

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
			}

			var cnt []byte
			if compact {
				cnt, err = json.Marshal(block)
			} else {
				cnt, err = json.MarshalIndent(block, "", " ")
			}
			if err != nil {
				return fmt.Errorf("unable to json marshall block: %w", err)
			}
			fmt.Println(string(cnt))
			return nil
		})
	}
}
