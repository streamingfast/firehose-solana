package reproc

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/streamingfast/sf-solana/types"

	"go.uber.org/zap"

	"github.com/streamingfast/dstore"

	pbsolana "github.com/streamingfast/sf-solana/types/pb/sol/type/v1"

	"github.com/streamingfast/bstream"
)

type Writer interface {
	BundleSize() uint64
	Write(blk *bstream.Block) error
	Flush(ctx context.Context) error
	Next() error
}

const BUNDLE_SIZE = 100

type BundleWriter struct {
	w                     bstream.BlockWriter
	startBlockNum         uint64
	exclusiveStopBlockNum uint64
	buf                   *bytes.Buffer
	store                 dstore.Store
	oneBlockSuffix        string
}

func NewBundleWriter(startBlockNum uint64, mergedBlockStore dstore.Store) (*BundleWriter, error) {
	buffer := bytes.NewBuffer(nil)
	blockWriter, err := bstream.GetBlockWriterFactory.New(buffer)
	if err != nil {
		return nil, fmt.Errorf("unable to get block writer: %w", err)
	}
	if startBlockNum%100 != 0 {
		return nil, fmt.Errorf("bundle needs a clean start block %%100")
	}

	return &BundleWriter{
		w:                     blockWriter,
		buf:                   buffer,
		startBlockNum:         startBlockNum,
		exclusiveStopBlockNum: startBlockNum + BUNDLE_SIZE,
		store:                 mergedBlockStore,
	}, nil
}

var errBundleComplete = errors.New("bundle complete")

func (w *BundleWriter) BundleSize() uint64 {
	return BUNDLE_SIZE
}
func (w *BundleWriter) Write(blk *bstream.Block) error {
	if blk.Num() >= w.exclusiveStopBlockNum {
		return errBundleComplete
	}

	if err := w.w.Write(blk); err != nil {
		return fmt.Errorf("failed to write bstream block: %w", err)
	}

	return nil
}

func (w *BundleWriter) Flush(ctx context.Context) error {
	filename := fileNameForBlocksBundle(w.startBlockNum)
	zlog.Info("flushing merged block files",
		zap.Uint64("start_block_num", w.startBlockNum),
		zap.Uint64("stop_block_num", w.exclusiveStopBlockNum),
		zap.String("filename", filename),
	)

	err := w.store.WriteObject(ctx, filename, w.buf)
	if err != nil {
		return fmt.Errorf("writing block buffer to store: %w", err)
	}
	return nil
}

func (w *BundleWriter) Next() (err error) {
	w.buf = bytes.NewBuffer(nil)
	w.w, err = bstream.GetBlockWriterFactory.New(w.buf)
	if err != nil {
		return fmt.Errorf("unable to get block writer: %w", err)
	}
	w.startBlockNum = w.exclusiveStopBlockNum
	w.exclusiveStopBlockNum = w.startBlockNum + BUNDLE_SIZE
	if w.startBlockNum%100 != 0 {
		panic("weird start block")
	}
	return nil
}

func (r *Reproc) saveBlock(ctx context.Context, parentNum uint64, blk *pbsolana.ConfirmedBlock, zlogger *zap.Logger) error {
	if tracer.Enabled() {
		zlogger.Debug("writing block to bundle")
	}

	block, err := types.BlockFromPBSolanaProto(blk)
	if err != nil {
		return fmt.Errorf("unable to convert block to proto: %w", err)
	}

	block.LibNum = parentNum
	err = r.writer.Write(block)
	if err == errBundleComplete {
		if err := r.writer.Flush(ctx); err != nil {
			return fmt.Errorf("unable to flush bundle: %w", err)
		}
		if err := r.writer.Next(); err != nil {
			return fmt.Errorf("unable to go to next bundle: %w", err)
		}

		if err := r.writer.Write(block); err != nil {
			return fmt.Errorf("unable to write blokc in new bundle: %w", err)
		}

		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to write block to bundle: %w", err)
	}
	return nil
}
func fileNameForBlocksBundle(blockNum uint64) string {
	return fmt.Sprintf("%010d", blockNum)
}
