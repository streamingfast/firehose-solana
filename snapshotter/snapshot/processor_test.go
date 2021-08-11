package snapshot

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testWriter struct {
	buf     *bytes.Buffer
	written int
}

func NewTestWriter() *testWriter {
	return &testWriter{
		buf: bytes.NewBuffer(nil),
	}
}

func (tw *testWriter) Write(p []byte) (int, error) {
	n, err := io.Copy(tw.buf, bytes.NewBuffer(p))

	zlog.Info("wrote", zap.Int64("bytes", n), zap.Error(err))
	tw.written += int(n)
	return int(n), err
}

func (tw *testWriter) Close() error {
	return nil
}

func testWriterFunc(filename string) (w io.WriteCloser) {
	return NewTestWriter()
}

type testFileReader struct {
	file *os.File
	read int
}

func (t *testFileReader) Read(p []byte) (int, error) {
	n, err := t.file.Read(p)

	zlog.Debug("read from file", zap.Int("bytes", n), zap.Error(err))
	t.read += n
	return n, err
}

func TestUncompress(t *testing.T) {
	t.Skip("requires local file")
	f, err := os.Open("/home/colin/rocksdb.tar.bz2")
	if err != nil {
		t.Error(err.Error())
	}
	defer func() {
		_ = f.Close()
	}()

	testReader := &testFileReader{file: f}

	err = uncompress(testReader, testWriterFunc)
	require.NoError(t, err)
}
