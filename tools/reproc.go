package tools

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/streamingfast/sf-solana/reproc"
	"go.uber.org/zap"

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

	if oneBlockFile {
		writer, err := reproc.NewBlockWriter(oneblockSuffix, store)
		if err != nil {
			return fmt.Errorf("unable to setup bundle writer: %w", err)
		}
		reprocClient, err := reproc.New(client, writer)
		if err != nil {
			return fmt.Errorf("unable to create reproc: %w", err)
		}
		return reprocClient.Launch(ctx, startBlockNum, endBlockNum)

	}

	if startBlockNum/100*100 != startBlockNum {
		return fmt.Errorf("writing merged files: must start on a 100-blocks boundary, not %d", startBlockNum)
	}
	if endBlockNum/100*100 != endBlockNum {
		return fmt.Errorf("writing merged files: must stop on a 100-blocks boundary, not %d", endBlockNum)
	}

	startRange := startBlockNum
	for {
		var endRange uint64
		startRange, endRange = findStartEndBlock(ctx, startRange, endBlockNum, store)
		zlog.Info("resolved next bundle boundaries", zap.Uint64("start_range", startRange), zap.Uint64("end_range", endRange))
		if startRange == endRange {
			zlog.Info("nothing to process, range is already covered")
			return nil
		}
		writer, err := reproc.NewBundleWriter(startRange, store)
		if err != nil {
			return fmt.Errorf("unable to setup bundle writer: %w", err)
		}
		reprocClient, err := reproc.New(client, writer)
		if err != nil {
			return fmt.Errorf("unable to create reproc: %w", err)
		}
		if err := reprocClient.Launch(ctx, startRange, endRange); err != nil {
			return err
		}

		if endRange >= endBlockNum {
			return nil
		}
	}
}

func findStartEndBlock(ctx context.Context, start, end uint64, store dstore.Store) (uint64, uint64) {

	errDone := errors.New("done")
	errComplete := errors.New("complete")

	var seenStart *uint64
	var seenEnd *uint64

	hasEnd := end >= 100

	err := store.WalkFrom(ctx, "", reproc.FilenameForBlocksBundle(start), func(filename string) error {
		num, err := strconv.ParseUint(filename, 10, 64)
		if err != nil {
			return err
		}
		if num < start { // user has decided to start its merger in the 'future'
			return nil
		}

		if num == start {
			seenStart = &num
			return nil
		}

		// num > start
		if seenStart == nil {
			seenEnd = &num
			return errDone // first block after a hole
		}

		// increment by 100
		if num == *seenStart+100 {
			if hasEnd && num == end-100 { // at end-100, we return immediately with errComplete, this will return (end, end)
				return errComplete
			}

			seenStart = &num
			return nil
		}

		seenEnd = &num
		return errDone
	})

	if err != nil && !errors.Is(err, errDone) {
		if errors.Is(err, errComplete) {
			return end, end
		}
		zlog.Error("got error walking store", zap.Error(err))
		return start, end
	}

	switch {
	case seenStart == nil && seenEnd == nil:
		return start, end // nothing was found
	case seenStart == nil:
		if *seenEnd > end {
			return start, end // blocks were found passed our range
		}
		return start, *seenEnd // we found some blocks mid-range
	case seenEnd == nil:
		return *seenStart + 100, end
	default:
		return *seenStart + 100, *seenEnd
	}

}
