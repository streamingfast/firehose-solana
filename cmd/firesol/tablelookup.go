package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/streamingfast/dstore"
	accountsresolver "github.com/streamingfast/firehose-solana/accountresolver"
	kvstore "github.com/streamingfast/kvdb/store"
	_ "github.com/streamingfast/kvdb/store/badger3"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func newProcessAddressLookupCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	return &cobra.Command{
		Use:   "process-address-lookup {store} {destination-store} {badger-db}",
		Short: "scan the blocks and process and extract the address lookup data",
		RunE:  processAddressLookupE(logger, tracer),
		Args:  cobra.ExactArgs(3),
	}
}

func processAddressLookupE(logger *zap.Logger, tracer logging.Tracer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		sourceStore, err := dstore.NewDBinStore(args[0])
		if err != nil {
			return fmt.Errorf("unable to create sourceStore: %w", err)
		}

		destinationStore, err := dstore.NewDBinStore(args[1])
		if err != nil {
			return fmt.Errorf("unable to create sourceStore: %w", err)
		}

		db, err := kvstore.New(args[2])
		if err != nil {
			return fmt.Errorf("unable to create sourceStore: %w", err)
		}

		//todo: discover cursor from kv
		cursor := accountsresolver.NewCursor(154655004, nil)
		fmt.Println("Default Cursor", cursor)
		processor := accountsresolver.NewProcessor("reproc", cursor, accountsresolver.NewKVDBAccountsResolver(db), logger)

		//todo: needs a destination sourceStore to write the merge blocks with the address lookup resolved
		err = processor.ProcessMergeBlocks(ctx, sourceStore, destinationStore)
		if err != nil {
			return fmt.Errorf("unable to process merge blocks: %w", err)
		}

		return nil
	}
}
