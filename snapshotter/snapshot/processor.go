package snapshot

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/abourget/llerrgroup"
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

		//Build a writer for the untar process.
		err = unCompress(rocksdbReader, func(fileName string) (w io.Writer, closer func() error) {
			//filePath is "rocksdb/001653.sst"
			dest := p.destinationSnapshotFolder + "/" + fileName
			zlog.Info("untaring file",
				zap.String("file_name", fileName),
				zap.String("destination_bucket", p.destinationBucket),
				zap.String("dest_file_name", dest))
			h := p.client.Bucket(p.destinationBucket).Object(dest)
			hw := h.NewWriter(ctx)
			hw.ContentType = "application/octet-stream"
			hw.CacheControl = "public, max-age=86400"

			return hw, func() error {
				return hw.Close()
			}
		})

		if err != nil {
			return fmt.Errorf("uncompressing rocked: %w", err)
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

func unCompress(compressedDataReader io.Reader, getWriter func(fileName string) (w io.Writer, closer func() error)) error {
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
				destWriter, closer := getWriter(target)
				// copy over contents
				size, err := io.Copy(destWriter, tr)
				if err != nil {
					return fmt.Errorf("uploadling target: %s: %w", target, err)
				}

				zlog.Info("target uploaded",
					zap.String("target", target),
					zap.Int64("size", size),
				)

				if err := closer(); err != nil {
					return fmt.Errorf("closing destination write to target: %s: %w", target, err)
				}
				zlog.Info("target closed",
					zap.String("target", target),
				)
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

	zlog.Info("decompression done.")

	return nil
}
