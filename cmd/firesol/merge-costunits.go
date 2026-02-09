package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"github.com/streamingfast/dstore"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewMergeCostUnitsCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge-cost-units <source> <source-costunits> <destination> <start> <stop>",
		Short: "merge-cost-units reads merged blocks from source, adds the 'costUnits' fields to the blocks and writes them to destination",
		Args:  cobra.ExactArgs(5),
		RunE:  getMergeCostUnitsRunner(logger),
	}

	return cmd
}

func getMergeCostUnitsRunner(rootLog *zap.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {

		ctx := cmd.Context()

		srcStore, err := dstore.NewDBinStore(args[0])
		if err != nil {
			return fmt.Errorf("unable to create source store: %w", err)
		}

		costUnitStore, err := dstore.NewDBinStore(args[1])
		if err != nil {
			return fmt.Errorf("unable to create cost unit store: %w", err)
		}

		destStore, err := dstore.NewDBinStore(args[2])
		if err != nil {
			return fmt.Errorf("unable to create destination store: %w", err)
		}

		start, err := strconv.ParseUint(args[3], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing start block num: %w", err)
		}
		stop, err := strconv.ParseUint(args[4], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing stop block num: %w", err)
		}

		logger := rootLog
		lastFileProcessed := ""
		startWalkFrom := fmt.Sprintf("%010d", start-(start%100))
		err = srcStore.WalkFrom(ctx, "", startWalkFrom, func(filename string) error {
			logger.Debug("checking merged block file", zap.String("filename", filename))

			startBlock := mustParseUint64(filename)

			if startBlock > stop {
				logger.Debug("stopping at merged block file above stop block", zap.String("filename", filename), zap.Uint64("stop", stop))
				return io.EOF
			}

			if startBlock+100 < start {
				logger.Debug("skipping merged block file below start block", zap.String("filename", filename))
				return nil
			}

			// Read blocks from source store
			rc, err := srcStore.OpenObject(ctx, filename)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", filename, err)
			}
			defer rc.Close()

			br, err := bstream.NewDBinBlockReader(rc)
			if err != nil {
				return fmt.Errorf("creating block reader: %w", err)
			}

			blocks := []*pbbstream.Block{}
			for {
				block, err := br.Read()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return fmt.Errorf("reading block from bundle %s: %w", filename, err)
				}

				blocks = append(blocks, block)
			}

			// Read blocks from cost unit store with the same filename
			rcCost, err := costUnitStore.OpenObject(ctx, filename)
			if err != nil {
				return fmt.Errorf("failed to open cost unit file %s: %w", filename, err)
			}
			defer rcCost.Close()

			brCost, err := bstream.NewDBinBlockReader(rcCost)
			if err != nil {
				return fmt.Errorf("creating cost unit block reader: %w", err)
			}

			costBlocks := []*pbbstream.Block{}
			for {
				block, err := brCost.Read()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return fmt.Errorf("reading block from cost unit bundle %s: %w", filename, err)
				}

				costBlocks = append(costBlocks, block)
			}

			// Verify block counts match
			if len(blocks) != len(costBlocks) {
				return fmt.Errorf("block count mismatch for file %s: source has %d blocks, cost unit has %d blocks", filename, len(blocks), len(costBlocks))
			}

			// Merge cost units from cost unit blocks into source blocks
			for k := 0; k < len(blocks); k++ {
				mergedBlock, err := mergeCostunits(blocks[k], costBlocks[k], logger)
				if err != nil {
					return fmt.Errorf("merging cost units for block %d: %w", blocks[k].Number, err)
				}
				blocks[k] = mergedBlock
			}

			if err := writeMergedBlocks(startBlock, destStore, blocks); err != nil {
				return fmt.Errorf("writing merged block %d: %w", startBlock, err)
			}

			lastFileProcessed = filename

			return nil
		})
		fmt.Printf("Last file processed: %s.dbin.zst\n", lastFileProcessed)

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		return nil
	}
}

func mergeCostunits(block *pbbstream.Block, costunitBlock *pbbstream.Block, logger *zap.Logger) (*pbbstream.Block, error) {
	b := &pbsol.Block{}
	err := block.Payload.UnmarshalTo(b)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling solana block %d: %w", block.Number, err)
	}

	// Unmarshal the cost unit block
	costUnitB := &pbsol.Block{}
	err = costunitBlock.Payload.UnmarshalTo(costUnitB)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling cost unit block %d: %w", costunitBlock.Number, err)
	}

	// Verify block numbers match
	if block.Number != costunitBlock.Number {
		return nil, fmt.Errorf("block numbers do not match: %d != %d", block.Number, costunitBlock.Number)
	}

	// Verify transaction counts match
	if len(b.Transactions) != len(costUnitB.Transactions) {
		return nil, fmt.Errorf("transaction count mismatch for block %d: %d != %d", block.Number, len(b.Transactions), len(costUnitB.Transactions))
	}

	// Copy cost units from each transaction
	costUnitsSet := 0
	for i := range b.Transactions {
		if b.Transactions[i].Meta != nil && costUnitB.Transactions[i].Meta != nil {
			costUnits := costUnitB.Transactions[i].Meta.CostUnits
			b.Transactions[i].Meta.CostUnits = costUnits
			if costUnits != nil {
				costUnitsSet++
			}
		}
	}

	if costUnitsSet > 0 {
		logger.Info("set cost units for block", zap.Uint64("block_num", block.Number), zap.Int("transactions", costUnitsSet))
	} else {
		logger.Info("no cost units for block", zap.Uint64("block_num", block.Number))
	}

	err = block.Payload.MarshalFrom(b)

	if err != nil {
		return nil, fmt.Errorf("marshaling solana block %d: %w", block.Number, err)
	}

	return block, nil
}

func writeMergedBlocks(lowBlockNum uint64, store dstore.Store, blocks []*pbbstream.Block) error {
	file := filename(lowBlockNum)
	fmt.Printf("writing merged file %s.dbin.zst\n", file)

	if len(blocks) == 0 {
		return fmt.Errorf("no blocks to write to bundle")
	}

	pr, pw := io.Pipe()

	go func() {
		var err error
		defer func() {
			pw.CloseWithError(err)
		}()

		blockWriter, err := bstream.NewDBinBlockWriter(pw)
		if err != nil {
			return
		}

		for _, blk := range blocks {
			err = blockWriter.Write(blk)
			if err != nil {
				return
			}
		}
	}()

	return store.WriteObject(context.Background(), file, pr)
}

func filename(num uint64) string {
	return fmt.Sprintf("%010d", num)
}

func mustParseUint64(in string) uint64 {
	out, err := strconv.ParseUint(in, 10, 64)
	if err != nil {
		panic(fmt.Errorf("unable to parse %q as uint64: %w", in, err))
	}

	return out
}
