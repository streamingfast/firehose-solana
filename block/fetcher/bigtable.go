package fetcher

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os/exec"
	"strings"
	"time"

	"cloud.google.com/go/bigtable"
	"github.com/klauspost/compress/zstd"
	pbsolv1 "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type BigtableBlockReader struct {
	bt             *bigtable.Client
	maxConnAttempt uint64

	logger *zap.Logger
	tracer logging.Tracer
}

func NewBigtableReader(bt *bigtable.Client, maxConnectionAttempt uint64, logger *zap.Logger, tracer logging.Tracer) *BigtableBlockReader {
	return &BigtableBlockReader{
		bt:             bt,
		logger:         logger,
		tracer:         tracer,
		maxConnAttempt: maxConnectionAttempt,
	}
}

var PrintFreq = uint64(10)

func (r *BigtableBlockReader) Read(
	ctx context.Context,
	startBlockNum,
	stopBlockNum uint64,
	processBlock func(block *pbsolv1.Block) error,
) error {
	var seenStartBlock bool
	var lastSeenBlock *pbsolv1.Block
	var fatalError error

	r.logger.Info("launching firehose-solana reprocessing",
		zap.Uint64("start_block_num", startBlockNum),
		zap.Uint64("stop_block_num", stopBlockNum),
	)
	table := r.bt.Open("blocks")
	attempts := uint64(0)

	for {
		if lastSeenBlock != nil {
			resolvedStartBlock := lastSeenBlock.GetFirehoseBlockNumber()
			r.logger.Debug("restarting read rows will retry last boundary",
				zap.Uint64("last_seen_block", lastSeenBlock.GetFirehoseBlockNumber()),
				zap.Uint64("resolved_block", resolvedStartBlock),
			)
			startBlockNum = resolvedStartBlock
		}

		btRange := bigtable.NewRange(fmt.Sprintf("%016x", startBlockNum), "")
		err := table.ReadRows(ctx, btRange, func(row bigtable.Row) bool {

			blk, zlogger, err := r.ProcessRow(row)
			if err != nil {
				fatalError = fmt.Errorf("failed to read row: %w", err)
				return false
			}

			if !seenStartBlock {
				if blk.Slot < startBlockNum {
					r.logger.Debug("skipping blow below start block",
						zap.Uint64("expected_block", startBlockNum),
					)
					return true
				}
				seenStartBlock = true
			}

			if lastSeenBlock != nil && lastSeenBlock.Blockhash == blk.Blockhash {
				r.logger.Debug("skipping block already seed",
					zap.Object("blk", blk),
				)
				return true
			}

			if lastSeenBlock != nil && (lastSeenBlock.Blockhash != blk.PreviousBlockhash) {
				// Weird cases where we do not receive the next linkeable block.
				// we should try to reconnect
				r.logger.Warn("received unlikable block",
					zap.Object("last_seen_blk", lastSeenBlock),
					zap.Object("blk", blk),
					zap.String("blk_previous_blockhash", blk.PreviousBlockhash),
				)
				return false
			}

			r.progressLog(blk, zlogger)
			lastSeenBlock = blk
			if err := processBlock(blk); err != nil {
				fatalError = fmt.Errorf("failed to write blokc: %w", err)
				return false
			}

			if stopBlockNum != 0 && blk.GetFirehoseBlockNumber() > stopBlockNum {
				return false
			}

			return true
		})

		if err != nil {
			attempts++
			if attempts >= r.maxConnAttempt {
				return fmt.Errorf("error while reading rowns, reached max attempts %d: %w", attempts, err)
			}
			r.logger.Error("error white reading rows", zap.Error(err), zap.Reflect("last_seen_block", lastSeenBlock), zap.Uint64("attempts", attempts))
			continue
		}
		if fatalError != nil {
			msg := "no blocks senn"
			if lastSeenBlock != nil {
				msg = fmt.Sprintf("last seen block %d (%s)", lastSeenBlock.GetFirehoseBlockNumber(), lastSeenBlock.GetFirehoseBlockID())
			}
			return fmt.Errorf("read blocks finished with a fatal error, %s: %w", msg, fatalError)
		}
		var opt []zap.Field
		if lastSeenBlock != nil {
			opt = append(opt, zap.Object("last_seen_block", lastSeenBlock))
		}
		r.logger.Debug("read block finished", opt...)
		if stopBlockNum != 0 {
			return nil
		}
		r.logger.Debug("stop block is num will sleep for 5 seconds and retry")
		time.Sleep(5 * time.Second)
	}
}

func (r *BigtableBlockReader) progressLog(blk *pbsolv1.Block, zlogger *zap.Logger) {
	if r.tracer.Enabled() {
		zlogger.Debug("handing block",
			zap.Uint64("parent_slot", blk.ParentSlot),
			zap.String("hash", blk.Blockhash),
		)
	}

	if blk.Slot%PrintFreq == 0 {
		opts := []zap.Field{
			zap.String("hash", blk.Blockhash),
			zap.String("previous_hash", blk.GetFirehoseBlockParentID()),
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

type RowType string

const (
	RowTypeProto RowType = "proto"
	RowTypeBin   RowType = "bin"
)

func explodeRow(row bigtable.Row) (*big.Int, RowType, []byte) {
	el := row["x"][0]
	var rowType RowType
	if strings.HasSuffix(el.Column, "proto") {
		rowType = RowTypeProto
	} else {
		rowType = RowTypeBin
	}
	blockNum, _ := new(big.Int).SetString(el.Row, 16)
	return blockNum, rowType, el.Value
}

func (r *BigtableBlockReader) ProcessRow(row bigtable.Row) (*pbsolv1.Block, *zap.Logger, error) {
	blockNum, rowType, rowCnt := explodeRow(row)
	zlogger := r.logger.With(
		zap.Uint64("block_num", blockNum.Uint64()),
		zap.String("row_type", string(rowType)),
		zap.String("row_key", row.Key()),
	)

	var cnt []byte
	var err error

	switch rowType {
	case RowTypeBin:
		cnt, err = externalBinToProto(rowCnt, "solana-bigtable-decoder", "--hex")
		if err != nil {
			return nil, zlogger, fmt.Errorf("unable get decode bin with external command 'solana-bigtable-decoder'  %s: %w", blockNum.String(), err)
		}
	default:
		cnt, err = r.decompress(rowCnt)
		if err != nil {
			return nil, zlogger, fmt.Errorf("unable to decompress block %s (uncompresse length %d): %w", blockNum.String(), len(rowCnt), err)
		}

	}
	zlogger.Debug("found bigtable row",
		zap.Stringer("blk_num", blockNum),
		zap.String("key", row.Key()),
		zap.Int("compressed_length", len(rowCnt)),
		zap.Int("uncompressed_length", len(cnt)),
		zap.String("row_type", string(rowType)),
	)

	blk := &pbsolv1.Block{}
	if err := proto.Unmarshal(cnt, blk); err != nil {
		return nil, zlogger, fmt.Errorf("unable to unmarshall confirmed block: %w", err)
	}
	blk.Slot = blockNum.Uint64()

	// horrible tweaks
	switch blk.Blockhash {
	case "Goi3t9JjgDkyULZbM2TzE5QqHP1fPeMcHNaXNFBCBv1v":
		zlogger.Warn("applying horrible tweak to block Goi3t9JjgDkyULZbM2TzE5QqHP1fPeMcHNaXNFBCBv1v")
		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
			blk.PreviousBlockhash = "HQEr9qcbUVBt2okfu755FdJvJrPYTSpzzmmyeWTj5oau"
		}
	case "6UFQveZ94DUKGbcLFoyayn1QwthVfD3ZqvrM2916pHCR":
		zlogger.Warn("applying horrible tweak to block 63,072,071")
		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
			blk.PreviousBlockhash = "7cLQx2cZvyKbGoMuutXEZ3peg3D21D5qbX19T5V1XEiK"
		}
	case "Fqbm7QvCTYnToXWcCw6nbkWhMmXx2Nv91LsXBrKraB43":
		zlogger.Warn("applying horrible tweak to block 53,135,959")
		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
			blk.PreviousBlockhash = "RfXUrekgajPSb1R4CGFJWNaHTnB6p53Tzert4gouj2u"
		}
	case "ABp9G2NaPzM6kQbeyZYCYgdzL8JN9AxSSbCQG2X1K9UF":
		zlogger.Warn("applying horrible tweak to block 46,223,993")
		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
			blk.PreviousBlockhash = "9F2C7TGqUpFu6krd8vQbUv64BskrneBSgY7U2QfrGx96"
		}
	case "ByUxmGuaT7iQS9qGS8on5xHRjiHXcGxvwPPaTGZXQyz7":
		zlogger.Warn("applying horrible tweak to block 61,328,766")
		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
			blk.PreviousBlockhash = "J6rRToKMK5DQDzVLqo7ibL3snwBYtqkYnRnQ7vXoUSEc"
		}
	}

	return blk, zlogger, nil
}

func externalBinToProto(in []byte, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to run command: %w", err)
	}

	inString := hex.EncodeToString(in)

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, inString)
	}()

	outCntHex, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed get command output: %w", err)
	}
	outHex := string(outCntHex)
	if strings.HasPrefix(outHex, "0x") {
		outHex = outHex[2:]
	}
	outHex = strings.TrimRight(outHex, "\n")
	cnt, err := hex.DecodeString(outHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode output string %q: %w", outHex, err)
	}
	return cnt, nil
}

func (r *BigtableBlockReader) decompress(in []byte) (out []byte, err error) {
	switch in[0] {
	case 0:
		r.logger.Debug("no compression found")
		out = in[4:]
	case 1:
		r.logger.Debug("bzip2 compression")
		// bzip2
		out, err = ioutil.ReadAll(bzip2.NewReader(bytes.NewBuffer(in[4:])))
		if err != nil {
			return nil, fmt.Errorf("bzip2 decompress: %w", err)
		}
	case 2:
		r.logger.Debug("gzip compression")
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
		r.logger.Debug("zstd compression")
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
