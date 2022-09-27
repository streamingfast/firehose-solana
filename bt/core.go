package bt

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigtable"
	pbsolv1 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v1"
	"go.uber.org/zap"
)

type Client struct {
	bt *bigtable.Client

	maxConnAttempt uint64
}

func New(bt *bigtable.Client, maxConnectionAttempt uint64) *Client {
	return &Client{
		bt:             bt,
		maxConnAttempt: maxConnectionAttempt,
	}
}

var PrintFreq = uint64(10)

func (r *Client) ReadBlocks(
	ctx context.Context,
	startBlockNum,
	stopBlockNum uint64,
	linkable bool,
	writer func(block *pbsolv1.Block) error,
) error {
	var seenStartBlock bool
	var lastSeenBlock *pbsolv1.Block
	var fatalError error

	zlog.Info("launching firehose-solana reprocessing",
		zap.Uint64("start_block_num", startBlockNum),
		zap.Uint64("stop_block_num", stopBlockNum),
	)
	table := r.bt.Open("blocks")
	attempts := uint64(0)

	for {
		if lastSeenBlock != nil {
			resolvedStartBlock := lastSeenBlock.Num()
			zlog.Debug("restarting read rows will retry last boundary",
				zap.Uint64("last_seen_block", lastSeenBlock.Num()),
				zap.Uint64("resolved_block", resolvedStartBlock),
			)
			startBlockNum = resolvedStartBlock
		}

		btRange := bigtable.NewRange(fmt.Sprintf("%016x", startBlockNum), "")
		err := table.ReadRows(ctx, btRange, func(row bigtable.Row) bool {

			blk, zlogger, err := processRow(row, zlog)
			if err != nil {
				fatalError = fmt.Errorf("failed to read row: %w", err)
				return false
			}

			if !seenStartBlock {
				if blk.Slot < startBlockNum {
					zlogger.Debug("skipping blow below start block",
						zap.Uint64("expected_block", startBlockNum),
					)
					return true
				}
				seenStartBlock = true
			}

			if lastSeenBlock != nil && lastSeenBlock.Blockhash == blk.Blockhash {
				zlogger.Debug("skipping block already seed",
					zap.Stringer("blk", blk.AsRef()),
				)
				return true
			}

			if lastSeenBlock != nil && linkable && (lastSeenBlock.Blockhash != blk.PreviousBlockhash) {
				// Weird cases where we do not receive the next linkeable block.
				// we should try to reconnect
				zlog.Warn("received unlikable block",
					zap.Stringer("last_seen_blk", lastSeenBlock.AsRef()),
					zap.Stringer("blk", blk.AsRef()),
					zap.String("blk_previous_blockhash", blk.PreviousBlockhash),
				)
				return false
			}

			progressLog(blk, zlogger)
			lastSeenBlock = blk
			if err := writer(blk); err != nil {
				fatalError = fmt.Errorf("failed to write blokc: %w", err)
				return false
			}

			if stopBlockNum != 0 && blk.Num() > stopBlockNum {
				return false
			}

			return true
		})

		if err != nil {
			attempts++
			if attempts >= r.maxConnAttempt {
				return fmt.Errorf("error while reading rowns, reached max attempts %d: %w", attempts, err)
			}
			zlog.Error("error white reading rows", zap.Error(err), zap.Reflect("last_seen_block", lastSeenBlock), zap.Uint64("attempts", attempts))
			continue
		}
		if fatalError != nil {
			msg := "no blocks senn"
			if lastSeenBlock != nil {
				msg = fmt.Sprintf("last seen block %q", lastSeenBlock.AsRef().String())
			}
			return fmt.Errorf("read blocks finished with a fatal error, %s: %w", msg, fatalError)
		}
		opt := []zap.Field{}
		if lastSeenBlock != nil {
			opt = append(opt, zap.Stringer("last_seen_block", lastSeenBlock.AsRef()))
		}
		zlog.Debug("read block finished", opt...)
		if stopBlockNum != 0 {
			return nil
		}
		zlog.Debug("stop block is num will sleep for 5 seconds and retry")
		time.Sleep(5 * time.Second)
	}
}

func progressLog(blk *pbsolv1.Block, zlogger *zap.Logger) {
	if tracer.Enabled() {
		zlogger.Debug("handing block",
			zap.Uint64("parent_slot", blk.ParentSlot),
			zap.String("hash", blk.Blockhash),
		)
	}

	if blk.Slot%PrintFreq == 0 {
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

		zlogger.Info(fmt.Sprintf("processing block 1 / %d", PrintFreq), opts...)
	}

}
