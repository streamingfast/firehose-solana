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
	"go.uber.org/zap"
)

const PRINT_FREQ = 10

type Reproc struct {
	bt             *bigtable.Client
	seenStartBlock bool

	writer        Writer
	lastSeenBlock *pbsolana.ConfirmedBlock
}

func New(bt *bigtable.Client, writer Writer) (*Reproc, error) {
	return &Reproc{
		bt:     bt,
		writer: writer,
	}, nil
}

func (r *Reproc) Launch(ctx context.Context, startBlockNum, stopBlockNum uint64) error {
	zlog.Info("launching sf-solana reprocessing",
		zap.Uint64("start_block_num", startBlockNum),
		zap.Uint64("start_block_num", stopBlockNum),
	)
	table := r.bt.Open("blocks")
	attempts := uint64(0)

	for {
		if r.lastSeenBlock != nil {
			resolvedStartBlock := r.lastSeenBlock.Num() - (r.lastSeenBlock.Num() % r.writer.BundleSize())
			zlog.Info("restarting read rows will retry last boundary",
				zap.Uint64("last_seen_block", r.lastSeenBlock.Num()),
				zap.Uint64("resolved_block", resolvedStartBlock),
				zap.Uint64("bundle_size", r.writer.BundleSize()),
			)
			startBlockNum = resolvedStartBlock
		}

		btRange := bigtable.NewRange(fmt.Sprintf("%016x", startBlockNum), fmt.Sprintf("%016x", stopBlockNum))
		err := table.ReadRows(ctx, btRange, func(row bigtable.Row) bool {
			return r.processRow(ctx, row, startBlockNum)
		})
		if err != nil {
			attempts++
			zlog.Error("error white reading rows", zap.Error(err), zap.Reflect("last_seen_block", r.lastSeenBlock), zap.Uint64("attempts", attempts))
			continue
		}
		zlog.Info("read block finished", zap.Reflect("last_seen_block", r.lastSeenBlock))
		return nil
	}

	return nil

}

func (r *Reproc) processRow(ctx context.Context, row bigtable.Row, startBlockNum uint64) bool {
	el := row["x"][0]
	blockNum, _ := new(big.Int).SetString(el.Row, 16)
	zlogger := zlog.With(zap.Uint64("block_num", blockNum.Uint64()))

	if !r.seenStartBlock {
		if blockNum.Uint64() < startBlockNum {
			zlogger.Warn("skipping blow below start block",
				zap.Uint64("expected_block", startBlockNum),
				zap.Uint64("received_block", blockNum.Uint64()),
			)
			return true
		}
		r.seenStartBlock = true
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
		return false
	}

	blk.Slot = blockNum.Uint64()

	if tracer.Enabled() {
		zlogger.Debug("handing block",
			zap.Uint64("slot", blk.Slot),
			zap.Uint64("parent_slot", blk.ParentSlot),
			zap.String("hash", blk.Blockhash),
		)
	}

	// Adjustment:
	// some blocks do not have a height in the proto bug, we assume
	// this is because the field was added later
	if blockNum.Uint64()%PRINT_FREQ == 0 {
		opts := []zap.Field{
			zap.String("hash", blk.Blockhash),
			zap.String("previous_hash", blk.PreviousID()),
			zap.Uint64("parent_slot", blk.ParentSlot),
		}

		if blk.BlockTime != nil {
			opts = append(opts, zap.Int64("timestamp", blk.BlockTime.Timestamp))
		} else {
			opts = append(opts, zap.Int64("timestamp", 0))
		}

		zlogger.Info(fmt.Sprintf("processing block 1 / %d", PRINT_FREQ), opts...)
	}
	if err := r.saveBlock(ctx, blk.ParentSlot, blk, zlogger); err != nil {
		zlogger.Warn("failed to write block", zap.Error(err))
		return false
	}
	r.lastSeenBlock = blk
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
