package reproc

import (
	"context"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/streamingfast/bstream"

	"go.uber.org/zap"

	"cloud.google.com/go/bigtable"

	"github.com/streamingfast/dstore"
	"github.com/streamingfast/merger"
	"github.com/streamingfast/merger/bundle"
)

type Filler struct {
	bt                *bigtable.Client
	startBlockNum     uint64
	store             *merger.DStoreIO
	zlogger           *zap.Logger
	mergedBlocksStore dstore.Store
	repocCli          *Reproc
}

func NewFiller(bt *bigtable.Client, startBlockNum uint64, oneBlockStore, mergedBlocksStore dstore.Store, client *Reproc, zlogger *zap.Logger) (*Filler, error) {
	return &Filler{
		bt:                bt,
		startBlockNum:     startBlockNum,
		mergedBlocksStore: mergedBlocksStore,
		store:             merger.NewDStoreIO(zlogger, nil, oneBlockStore, nil, 0, 0, bstream.GetProtocolFirstStreamableBlock, 100),
		zlogger:           zlogger,
		repocCli:          client,
	}, nil
}

var GetObjectTimeout = 5 * time.Minute

func (f *Filler) Run(ctx context.Context) error {
	f.zlogger.Info("filler running",
		zap.Uint64("start_block", f.startBlockNum),
		zap.Uint64("bundle_size", BUNDLE_SIZE),
	)
	bundler := bundle.NewBundler(zlog, f.startBlockNum, bstream.GetProtocolFirstStreamableBlock, BUNDLE_SIZE)
	err := bundler.Bootstrap(func(lowBlockNum uint64) ([]*bundle.OneBlockFile, error) {
		ctx, cancel := context.WithTimeout(context.Background(), GetObjectTimeout)
		defer cancel()
		reader, err := f.mergedBlocksStore.OpenObject(ctx, FilenameForBlocksBundle(f.startBlockNum-100))
		if err != nil {
			return nil, err
		}
		out, err := toOneBlockFile(reader)
		return out, err

	})
	if err != nil {
		return fmt.Errorf("bundle bootstrap: %w", err)
	}
	for {
		zlog.Info("bundle is not complete, re-walking", zap.Reflect("bundler", bundler))
		if err := f.retrieveOneBlockFile(ctx, bundler); err != nil {
			return err
		}
		isBundleComplete, highestBundleBlockNum, err := bundler.BundleCompleted()
		if err != nil {
			return err
		}
		if isBundleComplete {
			bundler.Commit(highestBundleBlockNum)
		} else {
			lb := bundler.LongestChainLastBlockFile()
			if lb == nil {
				return fmt.Errorf("bundle is not complete unable ot get last block to fill whole")
			}
			if err := f.repocCli.Launch(ctx, lb.Num, lb.Num+4); err != nil {
				return fmt.Errorf("failed to fil whole: %w", err)
			}
		}
	}
	return nil

}
func (f *Filler) retrieveOneBlockFile(ctx context.Context, bundler *bundle.Bundler) error {
	var highestSeenBlockFile *bundle.OneBlockFile
	seenFileCount := 0
	addedFileCount := 0
	callback := func(o *bundle.OneBlockFile) error {
		highestSeenBlockFile = o
		if bundler.IsBlockTooOld(o.Num) {
			return nil
		}
		exists := bundler.AddOneBlockFile(o)
		if exists {
			seenFileCount += 1
			return nil
		}
		addedFileCount++
		if addedFileCount >= 102 {
			return dstore.StopIteration
		}
		return nil
	}

	err := f.store.WalkOneBlockFiles(ctx, callback)
	if err != nil {
		return fmt.Errorf("fetching one block files: %w", err)
	}

	lowestBlock := bundler.BundleInclusiveLowerBlock()
	highest := bundler.LongestChainLastBlockFile()
	zapFields := []zap.Field{
		zap.Int("seen_files_count", seenFileCount),
		zap.Int("added_files_count", addedFileCount),

		zap.Uint64("lowest_block", lowestBlock),
		zap.Uint64("highest_linkable_block_file", highest.Num),
	}

	if highestSeenBlockFile != nil {
		zapFields = append(zapFields, zap.Uint64("highest_seen_block_file", highestSeenBlockFile.Num))
	}

	zlog.Info("retrieved list of files", zapFields...)
	return nil
}

// TODO(froch, 20220107): remove this code, dead code with new mindreader and correct filenames
func toOneBlockFile(mergeFileReader io.ReadCloser) (oneBlockFiles []*bundle.OneBlockFile, err error) {
	defer mergeFileReader.Close()

	blkReader, err := bstream.GetBlockReaderFactory.New(mergeFileReader)
	if err != nil {
		return nil, err
	}

	lowerBlock := uint64(math.MaxUint64)
	highestBlock := uint64(0)
	for {
		block, err := blkReader.Read()
		if block != nil {
			if block.Num() < lowerBlock {
				lowerBlock = block.Num()
			}

			if block.Num() > highestBlock {
				highestBlock = block.Num()
			}

			// we do this little dance to ensure that the 'canonical filename' will match any other oneblockfiles
			// the oneblock encoding/decoding stay together inside 'bundle' package
			fileName := bundle.BlockFileName(block)
			oneBlockFile := bundle.MustNewOneBlockFile(fileName)
			oneBlockFile.Merged = true
			oneBlockFiles = append(oneBlockFiles, oneBlockFile)
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return
}
