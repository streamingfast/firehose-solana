package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/streamingfast/dstore"
	accountsresolver "github.com/streamingfast/firehose-solana/accountresolver"
	kvstore "github.com/streamingfast/kvdb/store"
	_ "github.com/streamingfast/kvdb/store/badger3"
)

var processAddressLookupCmd = &cobra.Command{
	Use:   "process-address-lookup",
	Short: "scan the blocks and process and extract the address lookup data",
	RunE:  processAddressLookupE,
}

func processAddressLookupE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	store, err := dstore.NewDBinStore("gs://dfuseio-global-blocks-uscentral/sol-mainnet/v1?project=dfuseio-global")
	if err != nil {
		return fmt.Errorf("unable to create store: %w", err)
	}

	db, err := kvstore.New("badger3:///tmp/my-badger.db")
	if err != nil {
		return fmt.Errorf("unable to create store: %w", err)
	}

	cursor := accountsresolver.NewCursor(154655004, nil)
	fmt.Println("Default Cursor", cursor)
	processor := accountsresolver.NewProcessor("reproc", cursor, accountsresolver.NewKVDBAccountsResolver(db))

	//todo: needs a destination store to write the merge blocks with the address lookup resolved
	err = processor.ProcessMergeBlocks(ctx, store)
	if err != nil {
		return fmt.Errorf("unable to process merge blocks: %w", err)
	}

	return nil
}
