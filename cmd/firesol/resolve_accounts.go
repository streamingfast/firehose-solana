package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/mr-tron/base58"
	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
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

func newTrxAddressesLookupCmd(logger *zap.Logger, tracer logging.Tracer, chain *firecore.Chain[*pbsolv1.Block]) *cobra.Command {
	return &cobra.Command{
		Use:   "lookup-transaction {slot} {transaction} {blocks-store} {kv-dsn}",
		Short: "",
		RunE:  processTrxAddressesLookupE(chain, logger, tracer),
		Args:  cobra.ExactArgs(4),
	}
}

func processTrxAddressesLookupE(chain *firecore.Chain[*pbsolv1.Block], logger *zap.Logger, tracer logging.Tracer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		blockNum, err := strconv.ParseUint(args[0], 10, 64)

		bundleSlotNum := blockNum - (blockNum % 100)
		mergedBlocksFilename := fmt.Sprintf("%010d", bundleSlotNum)
		fmt.Println(mergedBlocksFilename)

		store, err := dstore.NewDBinStore(args[2])
		if err != nil {
			return fmt.Errorf("unable to create store at path %q: %w", store, err)
		}

		reader, err := store.OpenObject(ctx, mergedBlocksFilename)

		db, err := kvstore.New(args[3])
		if err != nil {
			return fmt.Errorf("unable to create sourceStore: %w", err)
		}

		readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
		if err != nil {
			fmt.Printf("❌ Unable to read blocks filename %s: %s\n", mergedBlocksFilename, err)
			return err
		}

		resolver := accountsresolver.NewKVDBAccountsResolver(db, logger)
		if err != nil {
			return fmt.Errorf("unable to parse start block number %s: %w", args[0], err)
		}

		for {
			block, err := readerFactory.Read()
			if err != nil {
				if err == io.EOF {
					return fmt.Errorf("block not found: %q", blockNum)
				}
				return fmt.Errorf("reading block: %w", err)
			}
			if blockNum == block.Num() {
				fmt.Println("Found block: ", blockNum)
				solBlock := block.ToProtocol().(*pbsolv1.Block)

				transactionToFind := args[1]
				for _, transaction := range solBlock.Transactions {
					sign := base58.Encode(transaction.Transaction.Signatures[0])
					if sign != transactionToFind {
						continue
					}
					fmt.Println("Found transaction: ", sign)
					addressToLookup := transaction.Transaction.Message.AddressTableLookups
					for _, a := range addressToLookup {
						address := accountsresolver.Account(a.AccountKey)
						accounts, _, err := resolver.Resolve(ctx, blockNum, address)
						if err != nil {
							return fmt.Errorf("unable to resolve account %s: %w", address, err)
						}
						fmt.Println("Address to lookup: ", address.Base58())
						for _, account := range accounts {
							fmt.Println(account.Base58())
						}
					}
					break
				}
				break
			}
		}

		fmt.Println("All done: Goodbye!")
		return nil
	}
}

func newAddressesLookupCmd(logger *zap.Logger, tracer logging.Tracer, chain *firecore.Chain[*pbsolv1.Block]) *cobra.Command {
	return &cobra.Command{
		Use:   "lookup {block} {table-address} {kv-dsn}",
		Short: "",
		RunE:  processAddressesLookupE(chain, logger, tracer),
		Args:  cobra.ExactArgs(3),
	}
}

func processAddressesLookupE(chain *firecore.Chain[*pbsolv1.Block], logger *zap.Logger, tracer logging.Tracer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		blockNum, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse block number %q: %w", args[0], err)
		}

		db, err := kvstore.New(args[2])
		if err != nil {
			return fmt.Errorf("unable to create sourceStore: %w", err)
		}

		resolver := accountsresolver.NewKVDBAccountsResolver(db, logger)
		if err != nil {
			return fmt.Errorf("unable to parse start block number %s: %w", args[0], err)
		}

		accs, _, err := resolver.Resolve(ctx, blockNum, accountsresolver.MustFromBase58(args[1]))
		if err != nil {
			return fmt.Errorf("unable to resolve account %s: %w", args[1], err)
		}

		for _, acc := range accs {
			fmt.Println(acc.Base58())
		}

		fmt.Println("All done: Goodbye!")
		return nil
	}
}
