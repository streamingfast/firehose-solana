package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streamingfast/dstore"
	firecore "github.com/streamingfast/firehose-core"
	accountsresolver "github.com/streamingfast/firehose-solana/accountresolver"
	pbsolv1 "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	kvstore "github.com/streamingfast/kvdb/store"
	_ "github.com/streamingfast/kvdb/store/badger3"
	_ "github.com/streamingfast/kvdb/store/bigkv"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func newProcessAddressLookupCmd(logger *zap.Logger, tracer logging.Tracer, chain *firecore.Chain[*pbsolv1.Block]) *cobra.Command {
	return &cobra.Command{
		Use:   "process-address-lookup {store} {destination-store} {badger-db}",
		Short: "scan the blocks and process and extract the address lookup data",
		RunE:  processAddressLookupE(chain, logger, tracer),
		Args:  cobra.ExactArgs(3),
	}
}

func processAddressLookupE(chain *firecore.Chain[*pbsolv1.Block], logger *zap.Logger, tracer logging.Tracer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			logger.Info("flag", zap.String("flag", flag.Name), zap.Reflect("value", flag.Value))
		})

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

		resolver := accountsresolver.NewKVDBAccountsResolver(db, logger)
		cursor, err := resolver.GetCursor(ctx, "reproc")
		if err != nil {
			return fmt.Errorf("unable to get cursor: %w", err)
		}

		if cursor == nil {
			logger.Info("No cursor found, starting from beginning")
			cursor = accountsresolver.NewCursor(154655004)
		}

		fmt.Println("Cursor", cursor)
		processor := accountsresolver.NewProcessor("reproc", cursor, resolver, logger)

		err = processor.ProcessMergeBlocks(ctx, sourceStore, destinationStore, chain.BlockEncoder)
		if err != nil {
			return fmt.Errorf("unable to process merge blocks: %w", err)
		}
		logger.Info("All done. Goodbye!")
		return nil
	}
}
