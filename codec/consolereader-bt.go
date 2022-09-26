package codec

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/firehose-solana/types"
	pbsolv1 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func NewBigtableConsoleReader(logger *zap.Logger, lines chan string) (*BigtableConsoleReader, error) {
	l := &BigtableConsoleReader{
		lines:  lines,
		close:  func() {},
		done:   make(chan interface{}),
		logger: logger,
	}
	return l, nil
}

type BigtableConsoleReader struct {
	lines  chan string
	close  func()
	done   chan interface{}
	logger *zap.Logger
}

func (cr *BigtableConsoleReader) ProcessData(reader io.Reader) error {
	scanner := cr.buildScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		cr.lines <- line
	}

	if scanner.Err() == nil {
		close(cr.lines)
		return io.EOF
	}

	return scanner.Err()
}

func (cr *BigtableConsoleReader) buildScanner(reader io.Reader) *bufio.Scanner {
	buf := make([]byte, 50*1024*1024)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(buf, 50*1024*1024)

	return scanner
}

func (cr *BigtableConsoleReader) Done() <-chan interface{} {
	return cr.done
}

func (cr *BigtableConsoleReader) Close() {
	cr.close()
}

func (cr *BigtableConsoleReader) ReadBlock() (out *bstream.Block, err error) {
	return cr.next()
}

func (cr *BigtableConsoleReader) next() (out *bstream.Block, err error) {
	for line := range cr.lines {
		if !strings.HasPrefix(line, "FIRE ") {
			continue
		}

		line = strings.TrimPrefix(line, "FIRE ") // removes the FIRE prefix
		blk, err := cr.parseLine(line)
		if err != nil {
			return nil, cr.formatError(line, err)
		}
		if blk != nil {
			return blk, nil
		}
	}
	cr.logger.Info("lines channel has been closed")
	return nil, io.EOF
}

func (cr *BigtableConsoleReader) parseLine(line string) (*bstream.Block, error) {
	if strings.HasPrefix(line, "BLOCK") {
		return cr.readBlock(line)
	}
	cr.logger.Warn("unable to handle log line. the log line may be known but the console reader may be in the wrong mod and cannot handle said log line",
		zap.String("line", line),
	)
	return nil, nil
}

func (cr *BigtableConsoleReader) formatError(line string, err error) error {
	chunks := strings.SplitN(line, " ", 2)
	return fmt.Errorf("%s: %s (line %q)", chunks[0], err, line)
}

//// BLOCK <SLOT_NUM> <COMPLETE BLOCK PROTO IN HEX>
func (cr *BigtableConsoleReader) readBlock(line string) (out *bstream.Block, err error) {
	cr.logger.Debug("reading block", zap.String("line", line))

	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != BlockCompleteChunk {
		return nil, fmt.Errorf("expected %d fields, got %d", BlockCompleteChunk, len(chunks))
	}

	var slotNum uint64
	if slotNum, err = strconv.ParseUint(chunks[1], 10, 64); err != nil {
		return nil, fmt.Errorf("slotNumber to int: %w", err)
	}

	var cnt []byte
	if cnt, err = hex.DecodeString(chunks[2]); err != nil {
		return nil, fmt.Errorf("unable to hex decode content: %w", err)
	}

	blk := &pbsolv1.Block{}
	if err := proto.Unmarshal(cnt, blk); err != nil {
		return nil, fmt.Errorf("unable to proto unmarhal confirmed block: %w", err)
	}
	blk.Slot = slotNum

	bstreamBlk, err := types.BlockFromPBSolanaProto(blk)
	if err != nil {
		return nil, fmt.Errorf("unable to convert solana proto block to bstream block: %w", err)
	}
	return bstreamBlk, nil
}
