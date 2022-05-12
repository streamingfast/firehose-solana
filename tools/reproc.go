package tools

import (
	"fmt"
	"strconv"

	"github.com/streamingfast/sf-solana/reproc"

	"cloud.google.com/go/bigtable"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dstore"
)

var reprocCmd = &cobra.Command{
	Use:   "reproc [project_id] [instance_id] [start_block] [end_block]",
	Short: "Download ConfirmedBlock objects from BigTable, and transform into merged blocks files",
	Args:  cobra.ExactArgs(4),
	RunE:  reprocRunE,
}

func init() {
	Cmd.AddCommand(reprocCmd)
	reprocCmd.Flags().String("oneblock-suffix", "default", "If non-empty, the oneblock files will be appended with that suffix, so that mindreaders can each write their file for a given block instead of competing for writes.")
	reprocCmd.Flags().String("dest-store", "./localblocks", "Destination blocks store")
	reprocCmd.Flags().Bool("one-block-files", false, "Generate one block files")
}

func reprocRunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	oneBlockFile := viper.GetBool("one-block-files")
	oneblockSuffix := viper.GetString("oneblock-suffix")
	store, err := dstore.NewDBinStore(viper.GetString("dest-store"))
	if err != nil {
		return fmt.Errorf("unable to create store at path %q: %w", store, err)
	}

	client, err := bigtable.NewClient(ctx, args[0], args[1])
	if err != nil {
		return err
	}

	startBlockNum, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse start block number %q: %w", args[2], err)
	}

	endBlockNum, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse end block number %q: %w", args[3], err)
	}

	var writer reproc.Writer
	if oneBlockFile {
		writer, err = reproc.NewBlockWriter(oneblockSuffix, store)
		if err != nil {
			return fmt.Errorf("unable to setup bundle writer: %w", err)
		}
	} else {
		writer, err = reproc.NewBundleWriter(startBlockNum, store)
		if err != nil {
			return fmt.Errorf("unable to setup bundle writer: %w", err)
		}
	}

	reprocClient, err := reproc.New(client, writer)
	if err != nil {
		return fmt.Errorf("unable to create reproc: %w", err)
	}
	return reprocClient.Launch(ctx, startBlockNum, endBlockNum)
}
