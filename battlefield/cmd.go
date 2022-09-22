package battlefield

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/streamingfast/firehose-solana/codec"
	"io"
	"io/ioutil"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/firehose-solana/types"
	pbsolv1 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v1"
	pbsolv2 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v2"
	"github.com/streamingfast/jsonpb"
	sftools "github.com/streamingfast/sf-tools"
	"go.uber.org/zap"
)

var Cmd = &cobra.Command{Use: "battlefield", Short: "Battlefield binary"}

func init() {
	Cmd.AddCommand(generateCmd)
	Cmd.AddCommand(compareCmd)
	compareCmd.Flags().Bool("ignore-extra-blocks", false, "This will ignore extra blocks that may be found in other file (not the reference)")
}

var generateCmd = &cobra.Command{
	Use:   "generate <path_to_firelog.firelog> <output.json> [path-to-firehose-batch-files]",
	Short: "Generated pbsol or pbsolana blocks from firehose logs.",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		firelogInputFilePath := args[0]
		jsonFilePath := args[1]
		augmentedStack := viper.GetBool("global-augmented-mode")
		batchFilesPath := ""
		if augmentedStack {
			if len(args) <= 2 {
				return fmt.Errorf("you must specficy a firehose batch files path as a third argument when running in --augmented-mode mode")
			}
			batchFilesPath = args[2]
		}

		zlog.Info("running battlefield generate",
			zap.String("firelog_file_path", firelogInputFilePath),
			zap.String("batch_file_path", batchFilesPath),
			zap.String("json_file_path", jsonFilePath),
		)

		opts := []codec.ConsoleReaderOption{codec.KeepBatchFiles()}
		if batchFilesPath != "" {
			opts = append(opts, codec.WithBatchFilesPath(batchFilesPath))
		}

		parser := &DMParser{
			crFactory: func() (*codec.ConsoleReader, error) {
				return codec.NewConsoleReader(zlog, make(chan string, 10000), opts...)
			},
			blockDecoder: types.PBSolanaBlockDecoder,
			blockCaster: func(i interface{}) proto.Message {
				return i.(*pbsolv1.Block)
			},
		}
		if augmentedStack {
			parser.blockDecoder = types.PBSolBlockDecoder
			parser.blockCaster = func(i interface{}) proto.Message {
				return i.(*pbsolv2.Block)
			}
		}

		blocks, err := parser.readLogs(firelogInputFilePath)
		if err != nil {
			return fmt.Errorf("failed to read firehose logs %q: %w", firelogInputFilePath, err)
		}
		zlog.Info("read all blocks from firehose logs from file",
			zap.Int("block_count", len(blocks)),
			zap.String("file", firelogInputFilePath),
		)

		fmt.Printf("Writing blocks to disk %q...", jsonFilePath)
		if err = parser.writeBlocks(jsonFilePath, blocks); err != nil {
			return fmt.Errorf("failed to write blocks: %w", err)
		}

		return nil
	},
}

var compareCmd = &cobra.Command{
	Use:   "compare {reference_blocks.json} {other_blocks.json}",
	Short: "Compares 2 blocks file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		referenceBlocksFilePath := args[0]
		otherBlocksFilePath := args[1]
		ignoreExtraBlocks := viper.GetBool("ignore-extra-blocks")

		zlog.Info("running battlefield compare",
			zap.String("reference_blocks_file_path", referenceBlocksFilePath),
			zap.String("other_blocks_file_path", otherBlocksFilePath),
			zap.Bool("ignore_extra_blocks", ignoreExtraBlocks),
		)

		matched, err := sftools.CompareBlockFiles(referenceBlocksFilePath, otherBlocksFilePath, func(refCnt, otherCnt []byte) (interface{}, interface{}, error) {
			refStandardBlocks := []*pbsolv1.Block{}
			err := json.Unmarshal(refCnt, &refStandardBlocks)
			if err != nil {
				zlog.Debug("failed unmarshal to array of standard blocks")
				refAugmentedBlocks := []*pbsolv2.Block{}
				err := json.Unmarshal(refCnt, &refAugmentedBlocks)
				if err != nil {
					return nil, nil, fmt.Errorf("unable to decode reference blocks in either augmented to standard blocks: %w", err)
				}

				otherAugmentedBlocks := []*pbsolv2.Block{}
				if err := json.Unmarshal(otherCnt, &otherAugmentedBlocks); err != nil {
					return nil, nil, fmt.Errorf("unable to decode other blocks as augmented blocks: %w", err)
				}

				if ignoreExtraBlocks {
					otherAugmentedBlocks = otherAugmentedBlocks[:len(refAugmentedBlocks)]
				}
				return refAugmentedBlocks, otherAugmentedBlocks, nil

			}

			otherStandardBlocks := []*pbsolv1.Block{}
			if err := json.Unmarshal(otherCnt, &otherStandardBlocks); err != nil {
				return nil, nil, fmt.Errorf("unable to decode other blocks as standard blocks: %w", err)
			}

			if ignoreExtraBlocks {
				otherStandardBlocks = otherStandardBlocks[:len(refStandardBlocks)]
			}
			return refStandardBlocks, otherStandardBlocks, nil
		}, zlog)
		if err != nil {
			return fmt.Errorf("failed to compare blocks")
		}
		if !matched {
			os.Exit(1)
		}
		return nil
	},
}

type DMParser struct {
	crFactory    func() (*codec.ConsoleReader, error)
	blockDecoder func(blk *bstream.Block) (interface{}, error)
	blockCaster  func(interface{}) proto.Message
}

func (d *DMParser) readLogs(filePath string) ([]interface{}, error) {
	blocks := []interface{}{}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open firehose logs file %q: %w", filePath, err)
	}
	defer file.Close()

	reader, err := d.crFactory()
	if err != nil {
		return nil, fmt.Errorf("unable to create console reader: %w", err)
	}
	defer reader.Close()

	go reader.ProcessData(file)
	var lastBlockRead bstream.BlockRef

	for {
		el, err := reader.ReadBlock()
		if err == io.EOF {
			break
		}

		if err != nil {
			if lastBlockRead == nil {
				return nil, fmt.Errorf("unable to read first block from file %q: %w", filePath, err)
			} else {
				return nil, fmt.Errorf("	unable to read block from file %q, last block read was %s: %w", filePath, lastBlockRead, err)
			}

		}

		block, err := d.blockDecoder(el)
		if err != nil {
			return nil, fmt.Errorf("unable to to transform bstream.Block into solana pb block: %w", err)
		}
		lastBlockRead = el.AsRef()
		blocks = append(blocks, block)
	}

	return blocks, nil
}

func (d *DMParser) writeBlocks(outputFilePath string, blocks []interface{}) error {
	buffer := bytes.NewBuffer(nil)
	if _, err := buffer.WriteString("[\n"); err != nil {
		return fmt.Errorf("unable to write list start: %w", err)
	}

	blockCount := len(blocks)
	if blockCount > 0 {
		lastIndex := blockCount - 1
		for i, blk := range blocks {
			block := d.blockCaster(blk)
			out, err := jsonpb.MarshalIndentToString(block, "  ")
			if err != nil {
				return fmt.Errorf("unable to marshal block %q: %w", block, err)
			}

			if _, err = buffer.WriteString(out); err != nil {
				return fmt.Errorf("unable to write block: %w", err)
			}

			if i != lastIndex {
				if _, err = buffer.WriteString(",\n"); err != nil {
					return fmt.Errorf("to write block delimiter: %w", err)
				}
			}
		}
	}

	if _, err := buffer.WriteString("]\n"); err != nil {
		return fmt.Errorf("unable to write list end: %w", err)
	}

	var unormalizedStruct []interface{}
	if err := json.Unmarshal(buffer.Bytes(), &unormalizedStruct); err != nil {
		return fmt.Errorf("unable to unmarshal JSON for normalization: %w", err)
	}

	normalizedJSON, err := json.MarshalIndent(unormalizedStruct, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to normalize JSON: %w", err)
	}

	err = ioutil.WriteFile(outputFilePath, normalizedJSON, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to write file %q: %w", outputFilePath, err)
	}
	return nil
}
