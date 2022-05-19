package reproc

import (
	"bytes"
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
)

type BlockWriter struct {
	w              bstream.BlockWriter
	buf            *bytes.Buffer
	store          dstore.Store
	oneBlockSuffix string
	filename       string
}

func NewBlockWriter(oneBlockSuffix string, oneBlockStore dstore.Store) (*BlockWriter, error) {
	buffer := bytes.NewBuffer(nil)
	blockWriter, err := bstream.GetBlockWriterFactory.New(buffer)
	if err != nil {
		return nil, fmt.Errorf("unable to get block writer: %w", err)
	}
	return &BlockWriter{
		w:              blockWriter,
		buf:            buffer,
		oneBlockSuffix: oneBlockSuffix,
		store:          oneBlockStore,
	}, nil
}

func (w *BlockWriter) BundleSize() uint64 {
	return 1
}

func (w *BlockWriter) Write(blk *bstream.Block) error {
	if err := w.w.Write(blk); err != nil {
		return fmt.Errorf("failed to write bstream block: %w", err)
	}
	w.filename = blockFileNameWithSuffix(blk, w.oneBlockSuffix)
	return errBundleComplete
}

func (w *BlockWriter) Flush(ctx context.Context) error {
	zlog.Info("flushing merged block files",
		zap.String("filename", w.filename),
	)

	err := w.store.WriteObject(ctx, w.filename, w.buf)
	if err != nil {
		return fmt.Errorf("writing block buffer to store: %w", err)
	}
	return nil
}

func (w *BlockWriter) Next() (err error) {
	w.buf = bytes.NewBuffer(nil)
	w.w, err = bstream.GetBlockWriterFactory.New(w.buf)
	if err != nil {
		return fmt.Errorf("unable to get block writer: %w", err)
	}
	return nil
}

func blockFileNameWithSuffix(block *bstream.Block, suffix string) string {
	blockTime := block.Time()
	blockTimeString := fmt.Sprintf("%s.%01d", blockTime.Format("20060102T150405"), blockTime.Nanosecond()/100000000)

	blockID := block.ID()
	if len(blockID) > 8 {
		blockID = blockID[len(blockID)-8:]
	}

	previousID := block.PreviousID()
	if len(previousID) > 8 {
		previousID = previousID[len(previousID)-8:]
	}

	return fmt.Sprintf("%010d-%s-%s-%s-%d-%s", block.Num(), blockTimeString, blockID, previousID, block.LibNum, suffix)
}
