package accountsresolver

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	firehose_solana "github.com/streamingfast/firehose-solana"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
)

func init() {
	firehose_solana.TestingInitBstream()
}

func Test_ProcessBlock(t *testing.T) {
	oneHundredBlockFilepath := "./devel"

	store, err := dstore.NewDBinStore(oneHundredBlockFilepath)
	ctx := context.Background()
	prefix := "0154656900"
	var files []string
	err = store.Walk(ctx, prefix, func(filename string) (err error) {
		files = append(files, filename)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		reader, err := store.OpenObject(ctx, file)
		defer reader.Close()

		blockReader, err := bstream.GetBlockReaderFactory.New(reader)
		if err != nil {
			t.Fatal(fmt.Errorf("Unable to read blocks filename %s: %s\n", file, err))
		}

		for {
			block, err := blockReader.Read()
			if err != nil {
				if err == io.EOF {
					// do not thing and continue
				}
				t.Fatal(fmt.Errorf("reading block: %w", err))
			}

			blk := block.ToProtocol().(*pbsol.Block)
			processor := &Processor{}
			err = processor.ProcessBlock(context.Background(), blk)
			if err != nil {
				t.Fatal(fmt.Errorf("processing block %d: %w", blk.Slot, err))
			}
		}
	}
}

func Test_ProcessTransaction(t *testing.T) {

}
