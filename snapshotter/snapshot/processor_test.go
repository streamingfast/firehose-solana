package snapshot

import (
	"io"
	"os"
	"testing"

	"github.com/test-go/testify/require"
	"go.uber.org/zap"
)

func TestUnCompress(t *testing.T) {
	t.Skip("processing test")
	f, err := os.Open("/Users/cbillett/t/toto.tar.bz2")
	require.NoError(t, err)
	err = unCompress(f, func(fileName string) (w io.Writer, closer func() error) {
		//filePath is "rocksdb/001653.sst"
		dest := "/Users/cbillett/t/untared/" + fileName
		zlog.Info("untaring file",
			zap.String("file_name", fileName),
			zap.String("destination", dest))

		d, err := os.Create(dest)
		require.NoError(t, err)
		return d, func() error {
			return d.Close()
		}
	})

	require.NoError(t, err)
}
