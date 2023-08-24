package solana_accounts_resolver

import (
	"context"
	"fmt"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	"io"
	"testing"
)

func blockReaderFactory(reader io.Reader) (bstream.BlockReader, error) {
	return bstream.NewDBinBlockReader(reader, func(contentType string, version int32) error {
		if contentType != "SOL" && version != 1 {
			return fmt.Errorf("reader only knows about %s block kind at version 1, got %s at version %d", "SOL", contentType, version)
		}

		return nil
	})
}

func Test_ProcessBlock(t *testing.T) {
	oneHundredBlockFile := "./devel"

	bstream.GetBlockReaderFactory = bstream.BlockReaderFactoryFunc(blockReaderFactory)
	store, err := dstore.NewDBinStore(oneHundredBlockFile)
	ctx := context.Background()
	prefix := "0154656200"
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

		readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
		if err != nil {
			t.Fatal(fmt.Errorf("Unable to read blocks filename %s: %s\n", file, err))
		}

		for {
			block, err := readerFactory.Read()
			if err != nil {
				if err == io.EOF {
					t.Fatal(fmt.Errorf("block not found: %q", 10))
				}
				t.Fatal(fmt.Errorf("reading block: %w", err))
			}

			fmt.Println(block)
		}
	}
}

func Test_ProcessTransaction(t *testing.T) {

}
