package accountsresolver

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/streamingfast/bstream"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

//this is a ripoff of merger/bundler.go

type BundleReader struct {
	ctx              context.Context
	readBuffer       []byte
	readBufferOffset int
	blockData        chan []byte
	errChan          chan error
	logger           *zap.Logger
	headerWritten    bool

	lastRead time.Time
}

func NewBundleReader(ctx context.Context, logger *zap.Logger) *BundleReader {
	return &BundleReader{
		ctx:           ctx,
		headerWritten: false,
		blockData:     make(chan []byte, 1),
		errChan:       make(chan error, 1),
		logger:        logger,
		lastRead:      time.Now(),
	}
}

func (r *BundleReader) Close() {
	close(r.blockData)
}

func (r *BundleReader) PushError(err error) {
	r.errChan <- err
}

func (r *BundleReader) PushBlock(block *bstream.Block) error {
	protoBlock, err := block.ToProto()
	if err != nil {
		return fmt.Errorf("unable to convert block to proto: %w", err)
	}

	data, err := proto.Marshal(protoBlock)
	if err != nil {
		return fmt.Errorf("unable to marshal proto block: %w", err)
	}

	if !r.headerWritten {
		header := []byte{'d', 'b', 'i', 'n', byte(0), 's', 'o', 'l', 0, 1}
		r.blockData <- header
		r.headerWritten = true
	}

	select {
	case <-r.ctx.Done():
		return nil
	case r.blockData <- data[bstream.GetBlockWriterHeaderLen:]:
		return nil
	}
}

func (r *BundleReader) Read(p []byte) (bytesRead int, err error) {
	//r.logger.Debug("read called", zap.Duration("since_last_read", time.Since(r.lastRead)))
	if r.readBuffer == nil {
		if err := r.fillBuffer(); err != nil {
			return 0, err
		}
	}

	bytesRead = copy(p, r.readBuffer[r.readBufferOffset:])
	r.readBufferOffset += bytesRead
	if r.readBufferOffset >= len(r.readBuffer) {
		r.readBuffer = nil
	}

	r.lastRead = time.Now()
	return bytesRead, nil
}

func (r *BundleReader) fillBuffer() error {
	var data []byte
	select {
	case d, ok := <-r.blockData:
		if !ok && d == nil {
			return io.EOF
		}
		data = d
		if !ok {
			return io.EOF
		}
	case err := <-r.errChan:
		return err
	case <-r.ctx.Done():
		return nil
	}

	if len(data) == 0 {
		r.readBuffer = nil
		return fmt.Errorf("one-block-file corrupt: empty data")
	}

	if len(data) < bstream.GetBlockWriterHeaderLen {
		return fmt.Errorf("one-block-file corrupt: expected header size of %d, but file size is only %d bytes", bstream.GetBlockWriterHeaderLen, len(data))
	}

	data = data[bstream.GetBlockWriterHeaderLen:]
	r.readBuffer = data
	r.readBufferOffset = 0
	return nil
}
