package tools

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/streamingfast/firehose-solana/bt"

	"cloud.google.com/go/bigtable"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var bigtableCmd = &cobra.Command{
	Use:   "bt",
	Short: "Solana bigtable sub command",
}

var bigtableGetCmd = &cobra.Command{
	Use:   "get <block_num>",
	Short: "get a block from bigtable",
	RunE:  bigtableGetRunE,
}

func init() {
	bigtableCmd.PersistentFlags().String("bt-project", "mainnet-beta", "Bigtable project")
	bigtableCmd.PersistentFlags().String("bt-instance", "solana-ledger", "Bigtable instance")
	Cmd.AddCommand(bigtableCmd)
	bigtableCmd.AddCommand(bigtableGetCmd)
}

func mustGetString(cmd *cobra.Command, flagName string) string {
	val, err := cmd.Flags().GetString(flagName)
	if err != nil {
		panic(fmt.Sprintf("flags: couldn't find flag %q", flagName))
	}
	return val
}

func bigtableGetRunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	zlog.Info("retrieving from bigtable")
	client, err := bigtable.NewClient(ctx, mustGetString(cmd, "bt-project"), mustGetString(cmd, "bt-instance"))
	if err != nil {
		return err
	}
	startBlockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block number %q: %w", args[2], err)
	}
	endBlockNum := startBlockNum + 1
	fmt.Println("Looking for block: ", startBlockNum)
	btRange := bigtable.NewRange(fmt.Sprintf("%016x", startBlockNum), fmt.Sprintf("%016x", endBlockNum))
	table := client.Open("blocks")
	err = table.ReadRows(ctx, btRange, func(row bigtable.Row) bool {
		blk, err := bt.ProcessRow(row, zlog)
		if err != nil {
			zlog.Error("unable to read bigtable row", zap.Error(err))
			return false
		}
		fmt.Println("Found bigtable row")
		cnt, err := json.MarshalIndent(blk, "", " ")
		if err != nil {
			zlog.Error("unable to marhsal", zap.Error(err))
			return false
		}
		fmt.Println(string(cnt))
		return true
	})
	return err
}
