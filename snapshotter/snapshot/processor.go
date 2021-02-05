package snapshot

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/abourget/llerrgroup"

	"cloud.google.com/go/storage"
	"github.com/mholt/archiver/v3"
	"go.uber.org/zap"
)

type processor struct {
	destinationBucket string
	destinationPrefix string
	workingDir        string
}

func NewProcessor(destinationBucket string, destinationFolder string, workingDir string) *processor {
	return &processor{
		destinationBucket: destinationBucket,
		destinationPrefix: destinationFolder,
		workingDir:        workingDir,
	}
}

func (p *processor) processSnapshot(ctx context.Context, client *storage.Client, snapshot string, sourceBucket string) error {
	zlog.Info("processing snapshot", zap.String("snapshot", snapshot), zap.String("sourceBucket", sourceBucket))
	_, err := listFiles(ctx, client, sourceBucket, snapshot, p.handleFile)

	if err != nil {
		return fmt.Errorf("file listing: %w", err)
	}

	return nil
}

func paddedSnapshotName(fileName string) string {
	parts := strings.Split(fileName, "/")
	return fmt.Sprintf("%09s", parts[0])
}

func relativeFilePath(fileName string) string {
	parts := strings.Split(fileName, "/")
	return strings.Join(parts[1:], "/")
}

func (p *processor) handleFile(ctx context.Context, client *storage.Client, bucket string, filePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	snapshot := paddedSnapshotName(filePath)
	relativeFilePath := relativeFilePath(filePath)

	zlog.Info("Handling file", zap.String("src_file_path", filePath), zap.String("snapshot", snapshot), zap.String("relative_file_path", relativeFilePath))
	srcFileHandler := client.Bucket(bucket).Object(filePath)

	if strings.HasSuffix(relativeFilePath, "rocksdb.tar.bz2") {
		zlog.Info("processing rocksdb file", zap.String("relative_file_path", relativeFilePath))
		rocksdbReader, err := srcFileHandler.NewReader(ctx)
		if err != nil {
			return fmt.Errorf("reader from rockdb source handler: %w", err)
		}

		//Build a writer for the untar process.
		rocksdbDestinationPrefix := p.destinationPrefix + "/" + snapshot
		err = unCompress(rocksdbReader, func(fileName string) *storage.Writer {
			//filePath is "rocksdb/001653.sst"
			dest := rocksdbDestinationPrefix + "/" + fileName
			zlog.Info("untaring file", zap.String("file_name", fileName), zap.String("dest_file_name", dest))
			h := client.Bucket(p.destinationBucket).Object(dest)
			return h.NewWriter(ctx)
		})

		if err != nil {
			return fmt.Errorf("uncompressing rocked: %w", err)
		}
		zlog.Info("uncompressed file")

		//todo: upload all file under rocksdb
	} else if strings.HasPrefix(filePath, "snapshot-") {
		//copy the file from 1 bucket to the other
		destinationFile := p.destinationPrefix + "/" + filePath
		destFileHandler := client.Bucket(p.destinationBucket).Object(destinationFile)

		attrs, err := destFileHandler.CopierFrom(srcFileHandler).Run(ctx)
		if err != nil {
			return fmt.Errorf("copy file: %s: %w", filePath, err)
		}
		zlog.Info("File copied", zap.String("snapshot", filePath), zap.String("filePath", bucket), zap.String("destination_bucket", p.destinationBucket), zap.String("destination_file", destinationFile), zap.String("attr_name", attrs.Name))
	} else {
		zlog.Info("Ignoring file", zap.String("file_name", filePath))
		return nil
	}

	return nil
}

func unCompress(compressedDataReader io.Reader, getWriter func(fileName string) *storage.Writer) error {
	cIface, err := archiver.ByExtension("foo.bz2")
	if err != nil {
		return fmt.Errorf("archive by extention: %w", err)
	}

	c, ok := cIface.(archiver.Decompressor)
	if !ok {
		return fmt.Errorf("not a decompressor by extention")
	}
	piper, pipew := io.Pipe()

	eg := llerrgroup.New(1)
	eg.Go(func() error {
		tr := tar.NewReader(piper)
		for {
			zlog.Info("waiting for header")
			header, err := tr.Next()
			zlog.Info("got an header", zap.Reflect("header", header))
			switch {

			// if no more files are found return
			case err == io.EOF:
				return nil

			// return any other error
			case err != nil:
				return err

			// if the header is nil, just skip it (not sure how this happens)
			case header == nil:
				continue
			}

			// the target location where the dir/file should be created
			target := filepath.Join(header.Name)

			switch header.Typeflag {
			case tar.TypeReg:
				zlog.Info("untar file", zap.String("target", target))
				destWriter := getWriter(target)
				destWriter.ContentType = "application/octet-stream"
				destWriter.CacheControl = "public, max-age=86400"
				// copy over contents
				if _, err := io.Copy(destWriter, tr); err != nil {
					return fmt.Errorf("updaling target: %s: %w", target, err)
				}
				zlog.Info("target uploaded", zap.String("target", target))

				if err := destWriter.Close(); err != nil {
					return fmt.Errorf("closing destination write to target: %s: %w", target, err)
				}
				return err
			}
		}
	})

	err = c.Decompress(compressedDataReader, pipew)
	if err != nil {
		return fmt.Errorf("decompressing: %w", err)
	}

	if err := eg.Wait(); err != nil {
		// eg.Wait() will block until everything is done, and return the first error.
		return err
	}
	return nil
}
