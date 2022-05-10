package reproc

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"math/big"

	pbsolana "github.com/streamingfast/sf-solana/types/pb/sol/type/v1"
	"google.golang.org/protobuf/proto"

	"cloud.google.com/go/bigtable"
	"github.com/klauspost/compress/zstd"
	"github.com/streamingfast/dstore"
	"go.uber.org/zap"
)

const PRINT_FREQ = 10

type Reproc struct {
	bt             *bigtable.Client
	startBlockNum  uint64
	stopBlockNum   uint64
	seenStartBlock bool

	bundleWriter *BundleWriter
}

func New(mergedBlockStore dstore.Store, bt *bigtable.Client, startBlockNum, stopBlockNum uint64) (*Reproc, error) {
	bw, err := NewBundleWriter(startBlockNum, mergedBlockStore)
	if err != nil {
		return nil, fmt.Errorf("unable to setup bundle writer: %w", err)
	}
	return &Reproc{
		bt:            bt,
		startBlockNum: startBlockNum,
		stopBlockNum:  stopBlockNum,
		bundleWriter:  bw,
	}, nil
}

func (r *Reproc) Launch(ctx context.Context) error {
	zlog.Info("launching sf-solana reprocessing",
		zap.Uint64("start_block_num", r.startBlockNum),
		zap.Uint64("start_block_num", r.stopBlockNum),
	)
	table := r.bt.Open("blocks")
	btRange := bigtable.NewRange(fmt.Sprintf("%016x", r.startBlockNum), fmt.Sprintf("%016x", r.stopBlockNum))
	if err := table.ReadRows(ctx, btRange, func(row bigtable.Row) bool {
		return r.processRow(ctx, row)
	}); err != nil {
		return fmt.Errorf("error while reading rows: %w", err)
	}

	return nil

}

func (r *Reproc) processRow(ctx context.Context, row bigtable.Row) bool {
	el := row["x"][0]
	blockNum, _ := new(big.Int).SetString(el.Row, 16)
	zlogger := zlog.With(zap.Uint64("block_num", blockNum.Uint64()))

	if !r.seenStartBlock {
		if blockNum.Uint64() != r.startBlockNum {
			zlogger.Warn("expected to receive start block as first block",
				zap.Uint64("expected_block", r.startBlockNum),
				zap.Uint64("received_block", blockNum.Uint64()),
			)
			return false
		}
		r.seenStartBlock = true
	}

	if tracer.Enabled() {
		zlogger.Debug("handing block")
	}

	var cnt []byte
	var err error
	if cnt, err = decompress(el.Value); err != nil {
		zlogger.Warn("failed to decompress payload", zap.Error(err))
		return false
	}

	blk := &pbsolana.ConfirmedBlock{}
	if err := proto.Unmarshal(cnt, blk); err != nil {
		zlogger.Warn("failed to unmarshal block", zap.Error(err))
		return true
	}

	// Adjustment:
	// some blocks do not have a height in the proto bug, we assume
	// this is because the field was added later

	if blk.BlockHeight == nil {
		blk.BlockHeight = &pbsolana.BlockHeight{
			BlockHeight: blockNum.Uint64(),
		}
	}

	if blockNum.Uint64()%PRINT_FREQ == 0 {
		opts := []zap.Field{
			zap.String("hash", blk.Blockhash),
			zap.String("previous_hash", blk.PreviousID()),
			zap.Uint64("parent_slot", blk.ParentSlot),
			zap.Uint64("block_height", blk.BlockHeight.BlockHeight),
		}

		if blk.BlockTime != nil {
			opts = append(opts, zap.Int64("timestamp", blk.BlockTime.Timestamp))
		} else {
			opts = append(opts, zap.Int64("timestamp", 0))
		}

		zlogger.Info(fmt.Sprintf("processing block 1 / %d", PRINT_FREQ), opts...)
	}
	if err := r.saveBlock(ctx, blockNum.Uint64(), blk, zlogger); err != nil {
		zlogger.Warn("failed to write block", zap.Error(err))
		return true
	}

	return true
}
func decompress(in []byte) (out []byte, err error) {
	switch in[0] {
	case 0:
		// uncompressed
	case 1:
		// bzip2
		out, err = ioutil.ReadAll(bzip2.NewReader(bytes.NewBuffer(in[4:])))
		if err != nil {
			return nil, fmt.Errorf("bzip2 decompress: %w", err)
		}
	case 2:
		// gzip
		reader, err := gzip.NewReader(bytes.NewBuffer(in[4:]))
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		out, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("gzip decompress: %w", err)
		}
	case 3:
		// zstd
		var dec *zstd.Decoder
		dec, err = zstd.NewReader(nil)
		if err != nil {
			return nil, fmt.Errorf("zstd reader: %w", err)
		}
		out, err = dec.DecodeAll(in[4:], out)
		if err != nil {
			return nil, fmt.Errorf("zstd decompress: %w", err)

		}
	default:
		return nil, fmt.Errorf("unsupported compression scheme for a block %d", in[0])
	}
	return
}
