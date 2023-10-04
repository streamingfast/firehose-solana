package accountsresolver

import (
	"context"
	"encoding/binary"
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

	lastRead time.Time
}

func NewBundleReader(ctx context.Context, logger *zap.Logger) *BundleReader {
	cntType := []byte("sol")
	ver := []byte{'0', '1'}
	return &BundleReader{
		ctx:       ctx,
		blockData: make(chan []byte, 1),
		errChan:   make(chan error, 1),
		logger:    logger,
		lastRead:  time.Now(),

		readBuffer: []byte{'d', 'b', 'i', 'n', byte(0), cntType[0], cntType[1], cntType[2], ver[0], ver[1]},
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

	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(data)))

	select {
	case <-r.ctx.Done():
		return nil
	case r.blockData <- append(length, data...):
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

	//data = data[bstream.GetBlockWriterHeaderLen:]
	r.readBuffer = data
	r.readBufferOffset = 0
	return nil
}
