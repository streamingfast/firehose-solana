package snapshot

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/mholt/archiver/v3"
	"go.uber.org/zap"
)

type processor struct {
	sourceBucket              string
	sourceSnapshotFolder      string
	destinationBucket         string
	destinationSnapshotFolder string
	workingDir                string
	client                    *storage.Client
}

func NewProcessor(sourceBucket string, sourceSnapshotName string, destinationBucket string, destinationSnapshotsFolder string, workingDir string, client *storage.Client) *processor {
	return &processor{
		sourceBucket:              sourceBucket,
		sourceSnapshotFolder:      sourceSnapshotName,
		destinationBucket:         destinationBucket,
		destinationSnapshotFolder: destinationSnapshotsFolder + "/" + paddedSnapshotName(sourceSnapshotName),
		workingDir:                workingDir,
		client:                    client,
	}
}

func (p *processor) processSnapshot(ctx context.Context) error {
	zlog.Info("processing snapshot",
		zap.String("source_bucket", p.sourceBucket),
		zap.String("source_snapshot_folder", p.sourceSnapshotFolder),
		zap.String("destination_bucket", p.destinationBucket),
		zap.String("destination_snapshot_folder", p.destinationSnapshotFolder),
	)

	zlog.Info("listing file from source",
		zap.String("source_bucket", p.sourceBucket),
		zap.String("source_snapshot_folder", p.sourceSnapshotFolder),
	)
	_, err := listFiles(ctx, p.client, p.sourceBucket, p.sourceSnapshotFolder, p.handleFile)
	if err != nil {
		return fmt.Errorf("file listing: %w", err)
	}

	err = p.writeProcessCompleteMarker(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *processor) writeProcessCompleteMarker(ctx context.Context) error {
	completedMarkerFile := p.destinationSnapshotFolder + "/" + "completed.marker"
	o := p.client.Bucket(p.destinationBucket).Object(completedMarkerFile)
	w := o.NewWriter(ctx)
	defer w.Close()

	_, err := w.Write([]byte{0x1})
	if err != nil {
		return fmt.Errorf("writting completion marker : %w", err)
	}
	return nil
}

func (p *processor) CompletedSnapshot(ctx context.Context) (bool, error) {
	completedMarkerFile := p.destinationSnapshotFolder + "/" + "completed.marker"
	_, err := p.client.Bucket(p.destinationBucket).Object(completedMarkerFile).Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func paddedSnapshotName(fileName string) string {
	parts := strings.Split(fileName, "/")
	return fmt.Sprintf("%09s", parts[0])
}

func relativeFilePath(fileName string) string {
	parts := strings.Split(fileName, "/")
	return strings.Join(parts[1:], "/")
}

func (p *processor) handleFile(ctx context.Context, filePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	zlog.Info("Handling file",
		zap.String("src_snapshot_folder", p.sourceSnapshotFolder),
		zap.String("src_file_path", filePath),
		zap.String("destination_bucket", p.destinationBucket),
		zap.String("destination_snapshot_folder", p.destinationSnapshotFolder),
	)

	srcFileHandler := p.client.Bucket(p.sourceBucket).Object(filePath)
	zlog.Info("got file handler", zap.String("object_name", srcFileHandler.ObjectName()))

	if strings.HasSuffix(filePath, "rocksdb.tar.bz2") {
		zlog.Info("processing rocksdb file")
		rocksdbReader, err := srcFileHandler.NewReader(ctx)
		if err != nil {
			return fmt.Errorf("reader from rockdb source handler: %w", err)
		}
		defer rocksdbReader.Close()

		//Build a writer for the untar process.
		err = uncompress(rocksdbReader, func(fileName string) (w io.WriteCloser) {
			//filePath is "rocksdb/001653.sst"
			dest := p.destinationSnapshotFolder + "/" + fileName
			zlog.Info("untarring file",
				zap.String("file_name", fileName),
				zap.String("destination_bucket", p.destinationBucket),
				zap.String("dest_file_name", dest))
			h := p.client.Bucket(p.destinationBucket).Object(dest)
			hw := h.NewWriter(ctx)
			hw.ContentType = "application/octet-stream"
			hw.CacheControl = "public, max-age=86400"

			return hw
		})

		if err != nil {
			return fmt.Errorf("uncompressing rockdb: %w", err)
		}
		zlog.Info("uncompressed file")

	} else if strings.HasPrefix(relativeFilePath(filePath), "snapshot-") {
		//copy the file from 1 sourceBucket to the other
		destinationFile := p.destinationSnapshotFolder + "/" + relativeFilePath(filePath)
		destFileHandler := p.client.Bucket(p.destinationBucket).Object(destinationFile)
		_, err := destFileHandler.CopierFrom(srcFileHandler).Run(ctx)
		if err != nil {
			return fmt.Errorf("copy file: %s: %w", filePath, err)
		}
		zlog.Info("File copied",
			zap.String("file_path", filePath),
			zap.String("destination_bucket", p.destinationBucket),
			zap.String("destination_file", destinationFile),
		)
	} else {
		zlog.Info("Ignoring file", zap.String("file_name", filePath))
		return nil
	}

	return nil
}

func uncompress(sourceReader io.Reader, destinationWriterFunc func(fileName string) (w io.WriteCloser)) error {
	pr, pw := io.Pipe()

	readErrStream := make(chan error)
	go func() {
		var err error
		defer func() {
			_ = pw.Close()
			if err != nil && err != io.EOF {
				readErrStream <- err
			}
			close(readErrStream)
		}()

		archiverInterface, err := archiver.ByExtension("-.bz2")
		if err != nil {
			return
		}

		decompressor, ok := archiverInterface.(archiver.Decompressor)
		if !ok {
			err = fmt.Errorf("archiver does not satisfy interface")
			return
		}

		err = decompressor.Decompress(sourceReader, pw)
		return
	}()

	writeErrStream := make(chan error)
	go func() {
		var err error
		defer func() {
			_ = pr.CloseWithError(err)
			if err == io.EOF {
				zlog.Info("got eof")
			}

			if err != nil && err != io.EOF {
				writeErrStream <- err
			}
			close(writeErrStream)
		}()

		tarReader := tar.NewReader(pr)
	Out: // needed to break out of for-loop from inside switch statement
		for {
			var header *tar.Header
			header, err = tarReader.Next()
			switch {
			case err != nil:
				break Out
			case header == nil:
				err = fmt.Errorf("empty header")
				break Out
			}

			target := header.Name
			switch header.Typeflag {
			case tar.TypeReg:
				// done in func call here so that we can cleanly defer our Close() call inside this for-loop
				err = func() (copyError error) {
					wc := destinationWriterFunc(target)
					defer func() {
						err := wc.Close()
						if err != nil {
							copyError = err
						}
						zlog.Info("target closed", zap.String("target", target))
					}()

					var size int64
					size, copyError = io.Copy(wc, tarReader)
					if copyError != nil {
						return copyError
					}

					zlog.Info("target uploaded", zap.String("target", target), zap.Int64("size", size))
					return nil
				}()
			}
		}
	}()

	zlog.Info("waiting")

	err := <-writeErrStream
	if err != nil {
		return err
	}

	err = <-readErrStream
	if err != nil {
		return err
	}

	return nil
}
