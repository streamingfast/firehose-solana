package tools

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"strconv"

	"cloud.google.com/go/bigtable"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dstore"
)

var dlFromBtCmd = &cobra.Command{
	Use:   "download-from-bigtable [project_id] [instance_id] [start_block] [end_block]",
	Short: "Download ConfirmedBlock objects from BigTable, and transform into merged blocks files",
	Args:  cobra.ExactArgs(4),
	RunE:  dlFromBtCmdE,
}

func init() {
	Cmd.AddCommand(dlFromBtCmd)

	dlFromBtCmd.PersistentFlags().String("dest-store", "./localblocks", "Destination blocks store")
}

func dlFromBtCmdE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

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

	table := client.Open("blocks")
	table.ReadRows(ctx, bigtable.NewRange(fmt.Sprintf("%016x", startBlockNum), fmt.Sprintf("%016x", endBlockNum)), func(row bigtable.Row) bool {
		// spew.Dump(row)
		el := row["x"][0]
		fmt.Println("Block: ", el.Row)
		var cnt []byte
		if cnt, err = decompress(el.Value); err != nil {
			return false
		}

		fmt.Println("Block: ", el.Row)
		fmt.Println(" Content: ", len(cnt))

		// TODO: write that into the MERGER
		// Craft the envelope based on its contents

		return true
	})
	if err != nil {
		return fmt.Errorf("reading blocks: %w", err)
	}

	return nil
}
func decompress(in []byte) (out []byte, err error) {
	fmt.Println("Compression style", in[0])
	switch in[0] {
	case 0:
		// uncompressed
	case 1:
		// bzip2
		out, err = ioutil.ReadAll(bzip2.NewReader(bytes.NewBuffer(in[4:])))
	case 2:
		// gzip
		reader, err := gzip.NewReader(bytes.NewBuffer(in[4:]))
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		out, err = ioutil.ReadAll(reader)
	case 3:
		// zstd
		var dec *zstd.Decoder
		dec, err = zstd.NewReader(nil)
		if err != nil {
			return
		}
		out, err = dec.DecodeAll(in[4:], out)
	default:
		return nil, fmt.Errorf("unsupported compression scheme for a block %d", in[0])
	}
	return
}
