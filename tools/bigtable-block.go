package tools

import (
	"encoding/json"
	"fmt"
	"strconv"

	pbsolv1 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v1"

	"github.com/streamingfast/firehose-solana/bt"

	"cloud.google.com/go/bigtable"
	"github.com/spf13/cobra"
)

var bigtableBlockCmd = &cobra.Command{
	Use:   "block <block_num>",
	Short: "get a block from bigtable",
	RunE:  bigtableBlockRunE,
}

func init() {
	bigtableCmd.AddCommand(bigtableBlockCmd)
}

func mustGetString(cmd *cobra.Command, flagName string) string {
	val, err := cmd.Flags().GetString(flagName)
	if err != nil {
		panic(fmt.Sprintf("flags: couldn't find flag %q", flagName))
	}
	return val
}

func bigtableBlockRunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	zlog.Info("retrieving from bigtable")
	client, err := bigtable.NewClient(ctx, mustGetString(cmd, "bt-project"), mustGetString(cmd, "bt-instance"))
	if err != nil {
		return fmt.Errorf("unable to create big table client: %w", err)
	}

	btClient := bt.New(client, 10)

	startBlockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block number %q: %w", args[2], err)
	}
	endBlockNum := startBlockNum + 1
	fmt.Println("Looking for block: ", startBlockNum)

	foundBlock := false
	err := btClient.ReadBlocks(ctx, startBlockNum, endBlockNum, func(block *pbsolv1.Block) error {
		foundBlock = true
		fmt.Println("Found bigtable row")
		cnt, err := json.MarshalIndent(block, "", " ")
		if err != nil {
			return fmt.Errorf("unable to json marshal block: %w", err)
		}
		fmt.Println(string(cnt))
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find block %q: %w", startBlockNum, err)
	}
	if !foundBlock {
		fmt.Printf("Could not find desired block %d\n", startBlockNum)
	}
	return nil
}
