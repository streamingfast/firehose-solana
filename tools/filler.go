package tools

import (
	"fmt"
	"strconv"

	"github.com/spf13/viper"

	"github.com/streamingfast/firehose-solana/reproc"

	"cloud.google.com/go/bigtable"
	"github.com/spf13/cobra"
	"github.com/streamingfast/dstore"
)

var fillerCmd = &cobra.Command{
	Use:   "filler <project_id> <instance_id> <one_block_store> <merged_block_store> [startBlockNum]",
	Short: "Download ConfirmedBlock objects from BigTable, and transform into merged blocks files",
	Args:  cobra.MinimumNArgs(4),
	RunE:  fillerRunE,
}

func init() {
	fillerCmd.Flags().String("oneblock-suffix", "default", "If non-empty, the oneblock files will be appended with that suffix, so that readers can each write their file for a given block instead of competing for writes.")
	Cmd.AddCommand(fillerCmd)
}

func fillerRunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	oneblockSuffix := viper.GetString("oneblock-suffix")

	client, err := bigtable.NewClient(ctx, args[0], args[1])
	if err != nil {
		return err
	}

	store, err := dstore.NewDBinStore(args[2])
	if err != nil {
		return fmt.Errorf("unable to create store at path %q: %w", store, err)
	}

	mergedStore, err := dstore.NewDBinStore(args[3])
	if err != nil {
		return fmt.Errorf("unable to create store at path %q: %w", store, err)
	}

	startBlock := uint64(0)
	if len(args) > 4 {
		startBlock, err = strconv.ParseUint(args[4], 10, 64)
		if err != nil {
			return err
		}
	}
	writer, err := reproc.NewBlockWriter(oneblockSuffix, store)
	if err != nil {
		return fmt.Errorf("unable to setup bundle writer: %w", err)
	}

	reprocClient, err := reproc.New(client, writer)

	filler, err := reproc.NewFiller(client, startBlock, store, mergedStore, reprocClient, zlog)
	if err != nil {
		return fmt.Errorf("unable to create reproc: %w", err)
	}
	return filler.Run(ctx)
}
