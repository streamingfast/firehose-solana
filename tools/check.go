package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/jsonpb"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"go.uber.org/zap"
)

var errStopWalk = errors.New("stop walk")

var checkCmd = &cobra.Command{Use: "check", Short: "Various checks for deployment, data integrity & debugging"}

var checkMergedBlocksCmd = &cobra.Command{
	// TODO: Not sure, it's now a required thing, but we could probably use the same logic as `start`
	//       and avoid altogether passing the args. If this would also load the config and everything else,
	//       that would be much more seamless!
	Use:   "merged-blocks {store-url}",
	Short: "Checks for any holes in merged blocks as well as ensuring merged blocks integrity",
	Args:  cobra.ExactArgs(1),
	RunE:  checkMergedBlocksE,
}
var checkOneBlocksCmd = &cobra.Command{
	Use:   "one-blocks {store-url}",
	Short: "Checks for any holes in one blocks as well as ensuring merged blocks integrity",
	Args:  cobra.ExactArgs(1),
	RunE:  checkOneBlocksE,
}

func init() {
	Cmd.AddCommand(checkCmd)
	checkCmd.AddCommand(checkMergedBlocksCmd)
	checkCmd.AddCommand(checkOneBlocksCmd)

	checkCmd.PersistentFlags().StringP("range", "r", "", "Block range to use for the check, format is of the form '<start>:<stop>' (i.e. '-r 1000:2000')")

	checkMergedBlocksCmd.Flags().BoolP("print-stats", "s", false, "Natively decode each block in the segment and print statistics about it, ensuring it contains the required blocks")
	checkMergedBlocksCmd.Flags().BoolP("print-full", "f", false, "Natively decode each block and print the full JSON representation of the block, should be used with a small range only if you don't want to be overwhelmed")
}

type blockNum uint64

func (b blockNum) String() string {
	return "#" + strings.ReplaceAll(humanize.Comma(int64(b)), ",", " ")
}

func checkMergedBlocksE(cmd *cobra.Command, args []string) error {
	storeURL := args[0]
	fileBlockSize := uint32(100)

	fmt.Printf("Checking block holes on %s\n", storeURL)

	number := regexp.MustCompile(`(\d{10})`)

	var expected uint32
	var count int
	var baseNum32 uint32
	holeFound := false
	printIndividualSegmentStats := viper.GetBool("print-stats")
	printFullBlock := viper.GetBool("print-full")

	blockRange, err := getBlockRangeFromFlag()
	if err != nil {
		return err
	}

	expected = uint32(blockRange.Start)
	currentStartBlk := uint32(blockRange.Start)
	seenFilters := map[string]FilteringFilters{}

	blocksStore, err := dstore.NewDBinStore(storeURL)
	if err != nil {
		return err
	}

	ctx := context.Background()
	walkPrefix := walkBlockPrefix(blockRange, fileBlockSize)

	zlog.Debug("walking merged blocks", zap.Stringer("block_range", blockRange), zap.String("walk_prefix", walkPrefix))
	err = blocksStore.Walk(ctx, walkPrefix, ".tmp", func(filename string) error {
		match := number.FindStringSubmatch(filename)
		if match == nil {
			return nil
		}

		zlog.Debug("received merged blocks", zap.String("filename", filename))

		count++
		baseNum, _ := strconv.ParseUint(match[1], 10, 32)
		if baseNum+uint64(fileBlockSize) < blockRange.Start {
			zlog.Debug("base num lower then block range start, quitting")
			return nil
		}

		baseNum32 = uint32(baseNum)

		if printIndividualSegmentStats || printFullBlock {
			newSeenFilters := validateBlockSegment(blocksStore, filename, fileBlockSize, blockRange, printIndividualSegmentStats, printFullBlock)
			for key, filters := range newSeenFilters {
				seenFilters[key] = filters
			}
		}

		if baseNum32 != expected {
			// There is no previous valid block range if we are the ever first seen file
			if count > 1 {
				fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, roundToBundleEndBlock(expected-fileBlockSize, fileBlockSize))
			}

			fmt.Printf("❌ Missing blocks range %d - %d!\n", expected, roundToBundleEndBlock(baseNum32-fileBlockSize, fileBlockSize))
			currentStartBlk = baseNum32

			holeFound = true
		}
		expected = baseNum32 + fileBlockSize

		if count%10000 == 0 {
			fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, roundToBundleEndBlock(baseNum32, fileBlockSize))
			currentStartBlk = baseNum32 + fileBlockSize
		}

		if !blockRange.Unbounded() && roundToBundleEndBlock(baseNum32, fileBlockSize) >= uint32(blockRange.Stop-1) {
			return errStopWalk
		}

		return nil
	})
	if err != nil && err != errStopWalk {
		return err
	}

	actualEndBlock := roundToBundleEndBlock(baseNum32, fileBlockSize)
	if !blockRange.Unbounded() {
		actualEndBlock = uint32(blockRange.Stop)
	}

	fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, actualEndBlock)

	if len(seenFilters) > 0 {
		fmt.Println()
		fmt.Println("Seen filters")
		for _, filters := range seenFilters {
			fmt.Printf("- [Include %q, Exclude %q, System %q]\n", filters.Include, filters.Exclude, filters.System)
		}
		fmt.Println()
	}

	if holeFound {
		fmt.Printf("🆘 Holes found!\n")
	} else {
		fmt.Printf("🆗 No hole found\n")
	}

	return nil
}

func checkOneBlocksE(cmd *cobra.Command, args []string) error {
	storeURL := args[0]

	fmt.Printf("Checking for block holes on %s\n", storeURL)

	number := regexp.MustCompile(`(\d{10})`)

	var expected uint32
	var count int
	var baseNum32 uint32
	var prevBlockNum uint32
	holeFound := false

	blockRange, err := getBlockRangeFromFlag()
	if err != nil {
		return err
	}

	expected = uint32(blockRange.Start)
	currentStartBlk := uint32(blockRange.Start)
	seenFilters := map[string]FilteringFilters{}

	blocksStore, err := dstore.NewDBinStore(storeURL)
	if err != nil {
		return err
	}

	ctx := context.Background()
	walkPrefix := walkBlockPrefix(blockRange, 1)

	zlog.Debug("walking one blocks", zap.Stringer("block_range", blockRange), zap.String("walk_prefix", walkPrefix))
	err = blocksStore.Walk(ctx, walkPrefix, ".tmp", func(filename string) error {
		match := number.FindStringSubmatch(filename)
		if match == nil {
			return nil
		}

		count++
		baseNum, _ := strconv.ParseUint(match[1], 10, 32)
		baseNum32 = uint32(baseNum)

		zlog.Debug("received one blocks", zap.String("filename", filename), zap.Uint32("prev", prevBlockNum), zap.Uint64("base_num", baseNum), zap.Uint32("expected", expected), zap.Int64("diff", int64(baseNum32)-int64(expected)))
		if baseNum+uint64(1) < blockRange.Start {
			zlog.Debug("base num lower then block range start, quitting")
			return nil
		}

		if int64(baseNum32)-int64(expected) > 0 {
			// There is no previous valid block range if we are the ever first seen file
			if count > 1 {
				fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, expected-1)
			}

			fmt.Printf("❌ Missing blocks range %d - %d!\n", expected, baseNum32-1)
			currentStartBlk = baseNum32

			holeFound = true
		}
		if baseNum32 != prevBlockNum { //dont increment expected if when seeing duplicated block
			expected = baseNum32 + 1
		}

		prevBlockNum = baseNum32

		if count%10000 == 0 {
			fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, baseNum32)
			currentStartBlk = baseNum32 + 1
		}

		if !blockRange.Unbounded() && baseNum32 >= uint32(blockRange.Stop-1) {
			return errStopWalk
		}

		return nil
	})
	if err != nil && err != errStopWalk {
		return err
	}

	actualEndBlock := baseNum32
	if !blockRange.Unbounded() {
		actualEndBlock = uint32(blockRange.Stop)
	}

	fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, actualEndBlock)

	if len(seenFilters) > 0 {
		fmt.Println()
		fmt.Println("Seen filters")
		for _, filters := range seenFilters {
			fmt.Printf("- [Include %q, Exclude %q, System %q]\n", filters.Include, filters.Exclude, filters.System)
		}
		fmt.Println()
	}

	if holeFound {
		fmt.Printf("🆘 Holes found!\n")
	} else {
		fmt.Printf("🆗 No hole found\n")
	}

	return nil
}

func walkBlockPrefix(blockRange BlockRange, fileBlockSize uint32) string {
	if blockRange.Unbounded() {
		return ""
	}

	startString := fmt.Sprintf("%010d", roundToBundleStartBlock(uint32(blockRange.Start), fileBlockSize))
	endString := fmt.Sprintf("%010d", roundToBundleEndBlock(uint32(blockRange.Stop-1), fileBlockSize)+1)

	offset := 0
	for i := 0; i < len(startString); i++ {
		if startString[i] != endString[i] {
			return string(startString[0:i])
		}

		offset++
	}

	// At this point, the two strings are equal, to return the string
	return startString
}

func roundToBundleStartBlock(block, fileBlockSize uint32) uint32 {
	// From a non-rounded block `1085` and size of `100`, we remove from it the value of
	// `modulo % fileblock` (`85`) making it flush (`1000`).
	return block - (block % fileBlockSize)
}

func roundToBundleEndBlock(block, fileBlockSize uint32) uint32 {
	// From a non-rounded block `1085` and size of `100`, we remove from it the value of
	// `modulo % fileblock` (`85`) making it flush (`1000`) than adding to it the last
	// merged block num value for this size which simply `size - 1` (`99`) giving us
	// a resolved formulae of `1085 - (1085 % 100) + (100 - 1) = 1085 - (85) + (99)`.
	return block - (block % fileBlockSize) + (fileBlockSize - 1)
}

func validateBlockSegment(
	store dstore.Store,
	segment string,
	fileBlockSize uint32,
	blockRange BlockRange,
	printIndividualSegmentStats bool,
	printFullBlock bool,
) (seenFilters map[string]FilteringFilters) {
	reader, err := store.OpenObject(context.Background(), segment)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks segment %s: %s\n", segment, err)
		return
	}
	defer reader.Close()

	readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks segment %s: %s\n", segment, err)
		return
	}

	// FIXME: Need to track block continuity (100, 101, 102a, 102b, 103, ...) and report which one are missing
	seenBlockCount := 0
	for {
		block, err := readerFactory.Read()
		if block != nil {
			if !blockRange.Unbounded() {
				if block.Number >= blockRange.Stop {
					return
				}

				if block.Number < blockRange.Start {
					continue
				}
			}

			seenBlockCount++

			if printIndividualSegmentStats {
				payloadSize := len(block.PayloadBuffer)
				block := block.ToNative().(*pbcodec.Block)

				fmt.Printf("Block #%d (%s) (prev: %s) (%d bytes): %d transactions\n",
					block.Num(),
					block.ID(),
					block.PreviousId,
					payloadSize,
					len(block.Transactions),
				)
			}

			if printFullBlock {
				eosBlock := block.ToNative().(*pbcodec.Block)

				fmt.Printf(jsonpb.MarshalIndentToString(eosBlock, "  "))
			}

			continue
		}

		if block == nil && err == io.EOF {
			if seenBlockCount < expectedBlockCount(segment, fileBlockSize) {
				fmt.Printf("❌ Segment %s contained only %d blocks, expected at least 100\n", segment, seenBlockCount)
			}

			return
		}

		if err != nil {
			fmt.Printf("❌ Unable to read all blocks from segment %s after reading %d blocks: %s\n", segment, seenBlockCount, err)
			return
		}
	}
}

func expectedBlockCount(segment string, fileBlockSize uint32) int {
	// True only on EOSIO, on other chains, it's probably different from 1 to X
	if segment == "0000000000" {
		return int(fileBlockSize) - 2
	}

	return int(fileBlockSize)
}
