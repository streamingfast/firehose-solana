package battlefield

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"

	pbsolana "github.com/streamingfast/sf-solana/types/pb"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/jsonpb"
	"github.com/streamingfast/sf-solana/node-manager/codec"
	"github.com/streamingfast/sf-solana/types"
	pbsol "github.com/streamingfast/sf-solana/types/pb/sf/solana/type/v1"
	sftools "github.com/streamingfast/sf-tools"
	"go.uber.org/zap"
)

var Cmd = &cobra.Command{Use: "battlefield", Short: "Battlefield binary"}

func init() {
	Cmd.AddCommand(generateCmd)
	Cmd.AddCommand(compareCmd)
}

var generateCmd = &cobra.Command{
	Use:   "generate-augmented <path_to_dmlog.dmlog> <output.json> [path-to-deepmind-batch-files]",
	Short: "Generated pbsol or pbsolana blocks from dmlogs. If dmlogs",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dmlogInputFilePath := args[0]
		jsonFilePath := args[1]
		augmentedStack := viper.GetBool("global-augmented-mode")
		batchFilesPath := ""
		if augmentedStack && len(args) <= 2 {
			return fmt.Errorf("you must specficy a deepming batch files path as a third argument when running in --augmented-stack mode")
		} else {
			batchFilesPath = args[2]
		}

		zlog.Info("running battlefield generate",
			zap.String("dmlog_file_path", dmlogInputFilePath),
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
				return i.(*pbsolana.ConfirmedBlock)
			},
		}
		if augmentedStack {
			parser.blockDecoder = types.PBSolBlockDecoder
			parser.blockCaster = func(i interface{}) proto.Message {
				return i.(*pbsol.Block)
			}
		}

		blocks, err := parser.readLogs(dmlogInputFilePath)
		if err != nil {
			return fmt.Errorf("failed to read dmlogs %q: %w", dmlogInputFilePath, err)
		}
		zlog.Info("read all blocks from dmlog file",
			zap.Int("block_count", len(blocks)),
			zap.String("file", dmlogInputFilePath),
		)

		fmt.Printf("Writing blocks to disk %q...", jsonFilePath)
		if err = parser.writeBlocks(jsonFilePath, blocks); err != nil {
			return fmt.Errorf("failed to write blocks: %w", err)
		}

		return nil
	},
}

var compareCmd = &cobra.Command{
	Use:   "compare {reference_blocks.json} {blocks_b.json}",
	Short: "Compares 2 blocks file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		blockAFilePath := args[0]
		blockBFilePath := args[1]

		matched, err := sftools.CompareBlockFiles(blockAFilePath, blockBFilePath, zlog)
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
		return nil, fmt.Errorf("unable to open dmlof file %q: %w", filePath, err)
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
