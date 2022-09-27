package tools

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/viper"
	pbsolv1 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v1"
	"google.golang.org/protobuf/proto"

	"github.com/streamingfast/firehose-solana/bt"

	"cloud.google.com/go/bigtable"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var bigtableBlocksCmd = &cobra.Command{
	Use:   "blocks <start_block_num> <stop_block_num>",
	Short: "get a range of blocks from bigtable",
	RunE:  bigtableBlocksRunE,
}

func init() {
	bigtableCmd.AddCommand(bigtableBlocksCmd)
	bigtableBlocksCmd.Flags().Bool("firehose-enabled", false, "When enable the blocks read will output Firehose formated logs 'FIRE <block_num> <block_payload_in_hex>'")
	bigtableBlocksCmd.Flags().Bool("compact", false, "When printing in JSON it will print compact instead of pretty-printed output")
	bigtableBlocksCmd.Flags().Bool("linkable", false, "Ensure that no block is skipped they are linkeable")
}

func bigtableBlocksRunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	firehoseEnabled := viper.GetBool("firehose-enabled")
	compact := viper.GetBool("compact")
	linkable := viper.GetBool("linkable")
	zlog.Info("retrieving from bigtable",
		zap.Bool("firehose_enabled", firehoseEnabled),
		zap.Bool("compact", compact),
		zap.Bool("linkable", linkable),
	)
	client, err := bigtable.NewClient(ctx, mustGetString(cmd, "bt-project"), mustGetString(cmd, "bt-instance"))
	if err != nil {
		return err
	}
	startBlockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse start block number %q: %w", args[2], err)
	}

	stopBlockNum, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse stop block number %q: %w", args[2], err)
	}

	zlog.Info("reading bigtable blocks",
		zap.Uint64("start_block_num", startBlockNum),
		zap.Uint64("stop_block_num", stopBlockNum),
	)
	btClient := bt.New(client, 10)

	return btClient.ReadBlocks(ctx, startBlockNum, stopBlockNum, linkable, func(block *pbsolv1.Block) error {
		if firehoseEnabled {
			cnt, err := proto.Marshal(block)
			if err != nil {
				return fmt.Errorf("failed to proto  marshal pb sol block: %w", err)
			}

			fmt.Printf("FIRE BLOCK %d %s\n", block.Slot, hex.EncodeToString(cnt))
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
