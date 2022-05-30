package reproc

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigtable"
	"github.com/streamingfast/sf-solana/bt"
	pbsolv1 "github.com/streamingfast/sf-solana/types/pb/sf/solana/type/v1"
	"go.uber.org/zap"
)

const PRINT_FREQ = 10

type Reproc struct {
	bt             *bigtable.Client
	seenStartBlock bool

	writer        Writer
	lastSeenBlock *pbsolv1.Block
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
		zap.Uint64("stop_block_num", stopBlockNum),
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

		btRange := bigtable.NewRange(fmt.Sprintf("%016x", startBlockNum), "")
		err := table.ReadRows(ctx, btRange, func(row bigtable.Row) bool {
			return r.processRow(ctx, row, startBlockNum, stopBlockNum)
		})
		if err != nil {
			attempts++
			zlog.Error("error white reading rows", zap.Error(err), zap.Reflect("last_seen_block", r.lastSeenBlock), zap.Uint64("attempts", attempts))
			continue
		}
		zlog.Info("read block finished", zap.Stringer("last_seen_block", r.lastSeenBlock.AsRef()))
		return nil
	}

}

func (r *Reproc) processRow(ctx context.Context, row bigtable.Row, startBlockNum, stopBlockNum uint64) bool {
	blockNum, rowType, _ := bt.ExplodeRow(row)
	zlogger := zlog.With(zap.Uint64("block_num", blockNum.Uint64()), zap.String("row_type", string(rowType)))
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

	blk, err := bt.ProcessRow(row, zlogger)
	if err != nil {
		zlogger.Warn("failed to read row", zap.Error(err))
		return false
	}

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
	if stopBlockNum != 0 && blk.Num() > stopBlockNum { // means we wrote the bundle
		return false
	}
	return true
}
