package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"github.com/streamingfast/dstore"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

const mergedBlocksBundleSize = 100

func NewAddTransactionConfigsCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-transaction-configs <source> <complement> <destination>",
		Short: "add-transaction-configs writes the blocks of source to destination, with the transaction 'version' and 'transactionConfig' fields taken from complement",
		Long: `The source store is the source of truth for everything but the 'version' and
'transactionConfig' fields of each transaction message, which are taken from the block of the
same number in the complement store. Both stores must hold the same blocks, in bundles of 100
under the same filenames, each block holding the same transactions in the same order.

Whole bundles are written, so the range is widened to the bundle holding its first block and
the one holding its last.`,
		Args: cobra.ExactArgs(3),
		RunE: getAddTransactionConfigsRunner(logger),
	}

	cmd.Flags().Uint64P("start-block", "s", 0, "First block to process")
	cmd.Flags().Uint64P("stop-block", "t", 0, "Last block to process, 0 to run to the end of the source store")

	return cmd
}

func getAddTransactionConfigsRunner(rootLog *zap.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		sourceStore, err := dstore.NewDBinStore(args[0])
		if err != nil {
			return fmt.Errorf("unable to create source store: %w", err)
		}

		complementStore, err := dstore.NewDBinStore(args[1])
		if err != nil {
			return fmt.Errorf("unable to create complement store: %w", err)
		}

		destStore, err := dstore.NewDBinStore(args[2])
		if err != nil {
			return fmt.Errorf("unable to create destination store: %w", err)
		}

		start, err := cmd.Flags().GetUint64("start-block")
		if err != nil {
			return err
		}

		stop, err := cmd.Flags().GetUint64("stop-block")
		if err != nil {
			return err
		}

		if stop != 0 && stop < start {
			return fmt.Errorf("stop block %d is below start block %d", stop, start)
		}

		rootLog.Info("starting to add transaction configs",
			zap.String("source", args[0]),
			zap.String("complement", args[1]),
			zap.String("destination", args[2]),
			zap.Uint64("start", start),
			zap.Uint64("stop", stop),
		)

		var blocksProcessed, versionsSet, configsSet int
		lastFileProcessed := ""

		startWalkFrom := fmt.Sprintf("%010d", start-(start%mergedBlocksBundleSize))
		err = sourceStore.WalkFrom(ctx, "", startWalkFrom, func(filename string) error {
			startBlock := mustParseUint64(filename)

			if stop != 0 && startBlock > stop {
				rootLog.Debug("stopping at merged block file above stop block", zap.String("filename", filename), zap.Uint64("stop", stop))
				return io.EOF
			}

			if startBlock+mergedBlocksBundleSize < start {
				rootLog.Debug("skipping merged block file below start block", zap.String("filename", filename))
				return nil
			}

			blocks, err := readBundle(ctx, sourceStore, filename)
			if err != nil {
				return fmt.Errorf("reading source bundle: %w", err)
			}

			complementBlocks, err := readBundle(ctx, complementStore, filename)
			if err != nil {
				return fmt.Errorf("reading complement bundle: %w", err)
			}

			complementByNumber := make(map[uint64]*pbbstream.Block, len(complementBlocks))
			for _, block := range complementBlocks {
				complementByNumber[block.Number] = block
			}

			for i, block := range blocks {
				complement, found := complementByNumber[block.Number]
				if !found {
					return fmt.Errorf("block %d of bundle %s is missing from the complement store", block.Number, filename)
				}

				if block.Id != complement.Id {
					return fmt.Errorf("block %d is %s in the source store and %s in the complement store", block.Number, block.Id, complement.Id)
				}

				merged, versions, configs, err := addTransactionConfigs(block, complement)
				if err != nil {
					return fmt.Errorf("adding transaction configs to block %d: %w", block.Number, err)
				}

				blocks[i] = merged
				blocksProcessed++
				versionsSet += versions
				configsSet += configs
			}

			if err := writeMergedBlocks(startBlock, destStore, blocks); err != nil {
				return fmt.Errorf("writing merged block %d: %w", startBlock, err)
			}

			lastFileProcessed = filename

			return nil
		})
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		rootLog.Info("complete",
			zap.String("last_file_processed", lastFileProcessed),
			zap.Int("blocks_processed", blocksProcessed),
			zap.Int("versions_set", versionsSet),
			zap.Int("transaction_configs_set", configsSet),
		)

		return nil
	}
}

// addTransactionConfigs returns the source block with the 'version' and 'transactionConfig'
// of each transaction message replaced by the complement's, and reports how many of each it
// set. Everything else is the source's.
func addTransactionConfigs(block *pbbstream.Block, complement *pbbstream.Block) (out *pbbstream.Block, versionsSet int, configsSet int, err error) {
	sourceBlock := &pbsol.Block{}
	if err := block.Payload.UnmarshalTo(sourceBlock); err != nil {
		return nil, 0, 0, fmt.Errorf("unmarshaling source block: %w", err)
	}

	complementBlock := &pbsol.Block{}
	if err := complement.Payload.UnmarshalTo(complementBlock); err != nil {
		return nil, 0, 0, fmt.Errorf("unmarshaling complement block: %w", err)
	}

	if len(sourceBlock.Transactions) != len(complementBlock.Transactions) {
		return nil, 0, 0, fmt.Errorf("transaction count mismatch: source has %d, complement has %d", len(sourceBlock.Transactions), len(complementBlock.Transactions))
	}

	for i, transaction := range sourceBlock.Transactions {
		complementTransaction := complementBlock.Transactions[i]

		if !sameSignatures(transaction, complementTransaction) {
			return nil, 0, 0, fmt.Errorf("transaction %d is not the same in both stores", i)
		}

		message := transaction.GetTransaction().GetMessage()
		complementMessage := complementTransaction.GetTransaction().GetMessage()
		if message == nil || complementMessage == nil {
			return nil, 0, 0, fmt.Errorf("transaction %d has no message in the source store or in the complement store", i)
		}

		message.Version = complementMessage.Version
		message.TransactionConfig = complementMessage.TransactionConfig

		if message.Version != nil {
			versionsSet++
		}
		if message.TransactionConfig != nil {
			configsSet++
		}
	}

	if err := block.Payload.MarshalFrom(sourceBlock); err != nil {
		return nil, 0, 0, fmt.Errorf("marshaling block: %w", err)
	}

	return block, versionsSet, configsSet, nil
}

func sameSignatures(left *pbsol.ConfirmedTransaction, right *pbsol.ConfirmedTransaction) bool {
	leftSignatures := left.GetTransaction().GetSignatures()
	rightSignatures := right.GetTransaction().GetSignatures()

	if len(leftSignatures) != len(rightSignatures) {
		return false
	}

	for i, signature := range leftSignatures {
		if !bytes.Equal(signature, rightSignatures[i]) {
			return false
		}
	}

	return true
}

func readBundle(ctx context.Context, store dstore.Store, filename string) ([]*pbbstream.Block, error) {
	reader, err := store.OpenObject(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer reader.Close()

	blockReader, err := bstream.NewDBinBlockReader(reader)
	if err != nil {
		return nil, fmt.Errorf("creating block reader for %s: %w", filename, err)
	}

	var blocks []*pbbstream.Block
	for {
		block, err := blockReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading block from bundle %s: %w", filename, err)
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}
