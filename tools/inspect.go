package tools

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"

	pbsol "github.com/streamingfast/sf-solana/types/pb/sf/solana/type/v1"

	"github.com/mr-tron/base58"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect merged blocks, or a single block",
}

var inspectBlockCmd = &cobra.Command{
	Use:   "block {block_num}",
	Short: "Print the content summary of a  block",
	Args:  cobra.ExactArgs(1),
	RunE:  inspectBlockE,
}

var inspectBlocksCmd = &cobra.Command{
	Use:   "blocks {base_block_num}",
	Short: "Prints the content summary of a merged blocks file",
	Args:  cobra.ExactArgs(1),
	RunE:  inspectBlocksE,
}

var inspectBlocksGraphCmd = &cobra.Command{
	Use:   "range {start_block_num:stop_block}",
	Short: "Prints the content summary of a merged blocks file",
	Args:  cobra.ExactArgs(1),
	RunE:  inspectRangeE,
}

func init() {
	Cmd.AddCommand(inspectCmd)
	inspectCmd.PersistentFlags().String("store", "gs://dfuseio-global-blocks-us/sol-mainnet/v5", "block store")
	inspectCmd.PersistentFlags().Uint64("transactions-for-block", 0, "Include transaction IDs in output")
	inspectCmd.PersistentFlags().Bool("transactions", false, "Include transaction IDs in output")
	inspectCmd.PersistentFlags().Bool("instructions", false, "Include instruction output")

	inspectCmd.AddCommand(inspectBlockCmd)

	inspectCmd.AddCommand(inspectBlocksCmd)
	inspectBlocksCmd.Flags().Bool("viz", false, "Output .dot file")
	inspectBlocksCmd.Flags().Bool("data", false, "output block data statistic")

	inspectCmd.AddCommand(inspectBlocksGraphCmd)
}

func inspectRangeE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	number := regexp.MustCompile(`(\d{10})`)
	str := viper.GetString("store")
	store, err := dstore.NewDBinStore(str)
	if err != nil {
		return fmt.Errorf("unable to create store at path %q: %w", store, err)
	}
	fileBlockSize := uint32(100)

	blockRange, err := decodeBlockRange(args[0])
	if err != nil {
		return err
	}

	walkPrefix := walkBlockPrefix(blockRange, fileBlockSize)
	type virtualSlot struct {
		previousID string
		endID      string
		starNum    uint64
		endNum     uint64

		Id    string
		count uint64
	}
	virtualSlots := map[string]*virtualSlot{}

	fmt.Println("// Run: dot -Tpdf file.dot -o file.pdf")
	fmt.Println("digraph D {")

	err = store.Walk(ctx, walkPrefix, ".tmp", func(filename string) error {
		match := number.FindStringSubmatch(filename)
		if match == nil {
			return nil
		}

		baseNum, _ := strconv.ParseUint(match[1], 10, 32)
		if baseNum < (blockRange.Start - uint64(fileBlockSize)) {
			return nil
		}

		if baseNum > blockRange.Stop {
			return errStopWalk
		}

		reader, err := store.OpenObject(ctx, filename)
		if err != nil {
			return fmt.Errorf("unable to read blocks filename: %s: %w", filename, err)
		}
		defer reader.Close()

		readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
		if err != nil {
			return fmt.Errorf("unable to read block in file %s: %w", filename, err)
		}

		for {
			blk, err := readerFactory.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("reading block: %w", err)
			}

			if blk.Number < blockRange.Start {
				continue
			}

			block := blk.ToProtocol().(*pbsol.Block)

			isVirutal := false
			if block.Number != block.Number {
				isVirutal = true
			}

			blockIdBase58 := base58.Encode(block.Id)
			previousBlockIdBase58 := base58.Encode(block.PreviousId)

			currentID := fmt.Sprintf("%s%s", blockIdBase58[:8], blockIdBase58[len(blockIdBase58)-8:])
			previousID := fmt.Sprintf("%s%s", previousBlockIdBase58[:8], previousBlockIdBase58[len(previousBlockIdBase58)-8:])
			if !isVirutal {
				fmt.Printf(
					"  S%s [label=\"%s..%s\\n#%d t=%d\\nblk=%d lib=%d\"];\n  S%s -> S%s;\n",
					currentID,
					blockIdBase58[:8],
					blockIdBase58[len(blockIdBase58)-8:],
					block.Number,
					block.TransactionCount,
					block.Number,
					blk.LibNum,
					currentID,
					previousID,
				)
				continue
			}

			if vslot, found := virtualSlots[string(block.PreviousId)]; found {
				delete(virtualSlots, string(block.PreviousId))
				vslot.count++
				vslot.endID = blockIdBase58
				vslot.endNum = block.Number
				virtualSlots[string(block.Id)] = vslot
			} else {
				virtualSlots[string(block.Id)] = &virtualSlot{
					starNum:    block.Number,
					previousID: previousBlockIdBase58,
					endID:      blockIdBase58,
					count:      1,
				}
			}
		}
		return nil
	})

	for _, vslot := range virtualSlots {
		currentID := fmt.Sprintf("%s%s", vslot.endID[:8], vslot.endID[len(vslot.endID)-8:])
		previousID := fmt.Sprintf("%s%s", vslot.previousID[:8], vslot.previousID[len(vslot.previousID)-8:])

		fmt.Printf(
			"  S%s [label=\"%d virtual slots\\n# %d -> %d\"];\n  S%s -> S%s;\n",
			currentID,
			vslot.count,
			vslot.starNum,
			vslot.endNum,
			currentID,
			previousID,
		)
	}
	fmt.Println("}")
	return nil
}

func inspectBlocksE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	blockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block number %q: %w", args[0], err)
	}
	augmentedStack := viper.GetBool("global-augmented-mode")
	str := viper.GetString("store")

	store, err := dstore.NewDBinStore(str)
	if err != nil {
		return fmt.Errorf("unable to create store at path %q: %w", store, err)
	}

	readerFactory, closeFunc, fileURL, err := readMergedBlockFile(ctx, store, blockNum)
	if err != nil {
		return err
	}
	defer closeFunc()

	fmt.Printf("Merged Blocks File: %s\n", fileURL)

	seenBlockCount := 0
	for {
		block, err := readerFactory.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading block: %w", err)

		}

		if err := readBlock(block, augmentedStack); err != nil {
			return fmt.Errorf("processing block: %w", err)
		}

		seenBlockCount++
	}

	fmt.Printf("Total blocks: %d\n", seenBlockCount)
	return nil
}

func inspectBlockE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	blockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse block number %q: %w", args[0], err)
	}
	augmentedStack := viper.GetBool("global-augmented-mode")

	delta := blockNum % 100
	baseBlockNum := blockNum - delta

	str := viper.GetString("store")
	store, err := dstore.NewDBinStore(str)
	if err != nil {
		return fmt.Errorf("unable to create store at path %q: %w", store, err)
	}

	readerFactory, closeFunc, fileURL, err := readMergedBlockFile(ctx, store, baseBlockNum)
	if err != nil {
		return err
	}
	defer closeFunc()

	fmt.Printf("Merged Blocks File: %s\n", fileURL)
	for {
		block, err := readerFactory.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading block: %w", err)
		}
		if block.Number != blockNum {
			continue
		}

		if err := readBlock(block, augmentedStack); err != nil {
			return fmt.Errorf("processing block: %w", err)
		}

	}
	return nil
}

func readMergedBlockFile(ctx context.Context, blockStore dstore.Store, baseBlockNum uint64) (bstream.BlockReader, func(), string, error) {
	filename := fmt.Sprintf("%010d", baseBlockNum)
	reader, err := blockStore.OpenObject(ctx, filename)
	if err != nil {
		fmt.Printf("❌ Unable to read merge blocks filename %s: %s\n", filename, err)
		return nil, nil, "", err
	}

	readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks filename %s: %s\n", filename, err)
		return nil, nil, "", err
	}

	cleanUpFunc := func() {
		reader.Close()
	}
	return readerFactory, cleanUpFunc, blockStore.ObjectURL(filename), nil
}
