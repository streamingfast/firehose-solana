package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"

	"github.com/gagliardetto/solana-go"
	"github.com/spf13/cobra"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"github.com/streamingfast/dstore"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewRemoveVoteTransactionsCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-vote-transactions <source> <destination>",
		Short: "remove-vote-transactions writes the blocks of source to destination without their vote transactions",
		Long: `A vote transaction is one whose message lists the Vote program
(Vote111111111111111111111111111111111111111) in its static account keys, failed or not. This is
the same rule the 'blocks_without_votes' module of solana-common uses, except failed transactions
that are not votes are kept. The legacy 'payload_buffer' field, a second copy of the block that
older files carry next to 'payload', is dropped. Everything else in the block is written unchanged.

Whole bundles are written, so the range is widened to the bundle holding its first block and
the one holding its last.

The source files hold 100 blocks each. With '--target-bundle-size', the destination files hold
that many blocks instead, like 'firecore tools resize-merged-blocks' writes them. A file is
written once all of its blocks were read, so a last file whose blocks go past the end of the
source store is not written. The source files must follow each other without a missing file.`,
		Args: cobra.ExactArgs(2),
		RunE: getRemoveVoteTransactionsRunner(logger),
	}

	cmd.Flags().Uint64P("start-block", "s", 0, "First block to process")
	cmd.Flags().Uint64P("stop-block", "t", 0, "Last block to process, 0 to run to the end of the source store")
	cmd.Flags().Uint64("target-bundle-size", mergedBlocksBundleSize, "Number of blocks per merged-blocks file written to the destination store, a multiple of 100")

	return cmd
}

func getRemoveVoteTransactionsRunner(rootLog *zap.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		sourceStore, err := dstore.NewDBinStore(args[0])
		if err != nil {
			return fmt.Errorf("unable to create source store: %w", err)
		}

		destStore, err := dstore.NewDBinStore(args[1])
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

		targetBundleSize, err := cmd.Flags().GetUint64("target-bundle-size")
		if err != nil {
			return err
		}
		if targetBundleSize == 0 || targetBundleSize%mergedBlocksBundleSize != 0 {
			return fmt.Errorf("target bundle size %d must be a positive multiple of %d", targetBundleSize, mergedBlocksBundleSize)
		}

		bundler := &mergedBlocksBundler{
			bundleSize:   targetBundleSize,
			lowBlockNum:  start - start%targetBundleSize,
			stopBlockNum: math.MaxUint64,
			write: func(lowBlockNum uint64, blocks []*pbbstream.Block) error {
				return writeMergedBlocks(lowBlockNum, destStore, blocks)
			},
		}
		if stop != 0 {
			bundler.stopBlockNum = stop - stop%targetBundleSize + targetBundleSize
		}

		rootLog.Info("starting to remove vote transactions",
			zap.String("source", args[0]),
			zap.String("destination", args[1]),
			zap.Uint64("start", start),
			zap.Uint64("stop", stop),
			zap.Uint64("target_bundle_size", targetBundleSize),
		)

		var blocksProcessed, transactionsKept, votesRemoved int
		lastFileProcessed := ""

		startWalkFrom := filename(bundler.lowBlockNum - bundler.lowBlockNum%mergedBlocksBundleSize)
		err = sourceStore.WalkFrom(ctx, "", startWalkFrom, func(filename string) error {
			startBlock := mustParseUint64(filename)

			if startBlock >= bundler.stopBlockNum {
				rootLog.Debug("stopping at merged block file above stop block", zap.String("filename", filename), zap.Uint64("stop", stop))
				return io.EOF
			}

			if startBlock+mergedBlocksBundleSize <= bundler.lowBlockNum {
				rootLog.Debug("skipping merged block file below start block", zap.String("filename", filename))
				return nil
			}

			if lastFileProcessed == "" {
				// The source store can begin after the start block, the files below its first
				// one have no block to write
				bundler.lowBlockNum = max(bundler.lowBlockNum, startBlock-startBlock%targetBundleSize)
			} else if expected := mustParseUint64(lastFileProcessed) + mergedBlocksBundleSize; startBlock != expected {
				return fmt.Errorf("merged-blocks file %s is missing from the source store, found %s after %s", fmt.Sprintf("%010d", expected), filename, lastFileProcessed)
			}

			blocks, err := readBundle(ctx, sourceStore, filename)
			if err != nil {
				return fmt.Errorf("reading source bundle: %w", err)
			}

			for i, block := range blocks {
				out, kept, removed, err := removeVoteTransactions(block)
				if err != nil {
					return fmt.Errorf("removing vote transactions from block %d: %w", block.Number, err)
				}

				blocks[i] = out
				blocksProcessed++
				transactionsKept += kept
				votesRemoved += removed
			}

			if err := bundler.add(blocks); err != nil {
				return err
			}
			if err := bundler.writeBundlesEndingBefore(startBlock + mergedBlocksBundleSize); err != nil {
				return err
			}

			lastFileProcessed = filename

			if bundler.done() {
				return io.EOF
			}
			return nil
		})
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		if len(bundler.blocks) > 0 {
			rootLog.Warn("last merged-blocks file not written, the source store ends before its last block", zap.String("filename", filename(bundler.lowBlockNum)))
		}

		rootLog.Info("complete",
			zap.String("last_file_processed", lastFileProcessed),
			zap.Int("blocks_processed", blocksProcessed),
			zap.Int("transactions_kept", transactionsKept),
			zap.Int("vote_transactions_removed", votesRemoved),
		)

		return nil
	}
}

// removeVoteTransactions returns the block without its vote transactions, and reports how many
// transactions it kept and how many it removed.
func removeVoteTransactions(block *pbbstream.Block) (out *pbbstream.Block, kept int, removed int, err error) {
	solBlock := &pbsol.Block{}
	if err := block.Payload.UnmarshalTo(solBlock); err != nil {
		return nil, 0, 0, fmt.Errorf("unmarshaling block: %w", err)
	}

	before := len(solBlock.Transactions)
	solBlock.Transactions = slices.DeleteFunc(solBlock.Transactions, isVoteTransaction)
	kept = len(solBlock.Transactions)

	if err := block.Payload.MarshalFrom(solBlock); err != nil {
		return nil, 0, 0, fmt.Errorf("marshaling block: %w", err)
	}

	// Readers ignore 'payload_buffer' when 'payload' is set, and it still holds the vote transactions.
	block.PayloadBuffer = nil

	return block, kept, before - kept, nil
}

func isVoteTransaction(transaction *pbsol.ConfirmedTransaction) bool {
	for _, key := range transaction.GetTransaction().GetMessage().GetAccountKeys() {
		if bytes.Equal(key, solana.VoteProgramID[:]) {
			return true
		}
	}

	return false
}
