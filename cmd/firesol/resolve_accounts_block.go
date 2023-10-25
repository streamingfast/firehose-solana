package main

import (
	"context"
	"fmt"
	"github.com/hako/durafmt"
	"github.com/spf13/cobra"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dhammer"
	"github.com/streamingfast/dstore"
	firecore "github.com/streamingfast/firehose-core"
	accountsresolver "github.com/streamingfast/firehose-solana/accountresolver"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	kvstore "github.com/streamingfast/kvdb/store"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
	"time"
)

func newResolveAccountsBlockCmd(logger *zap.Logger, tracer logging.Tracer, chain *firecore.Chain[*pbsol.Block]) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve-accounts-block {store} {destination-store} {kv-dsn} {startBlock} {endBlock}",
		Short: "Apply table lookup accounts to merge blocks.",
		RunE:  processResolveAccountsBlockE(chain, logger, tracer),
		Args:  cobra.ExactArgs(5),
	}

	return cmd
}

func processResolveAccountsBlockE(chain *firecore.Chain[*pbsol.Block], logger *zap.Logger, tracer logging.Tracer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		srcStore, err := dstore.NewDBinStore(args[0])
		if err != nil {
			return fmt.Errorf("unable to create source store: %w", err)
		}

		destStore, err := dstore.NewDBinStore(args[1])
		if err != nil {
			return fmt.Errorf("unable to create destination store: %w", err)
		}

		db, err := kvstore.New(args[2])
		if err != nil {
			return fmt.Errorf("unable to create sourceStore: %w", err)
		}

		resolver := accountsresolver.NewKVDBAccountsResolver(db, logger)
		if err != nil {
			return fmt.Errorf("unable to get cursor: %w", err)
		}

		startBlock, err := strconv.ParseUint(args[3], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing start block: %w", err)
		}

		stopBlock, err := strconv.ParseUint(args[4], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing stop block: %w", err)
		}

		err = processMergeBlocks(ctx, startBlock, stopBlock, srcStore, destStore, resolver, chain.BlockEncoder, logger)
		if err != nil {
			return fmt.Errorf("processing merge blocks for range %d - %d: %w", startBlock, stopBlock, err)
		}

		logger.Info("All done. Goodbye!")
		return nil
	}
}

func processMergeBlocks(
	ctx context.Context,
	startBlock uint64,
	stopBlock uint64,
	sourceStore dstore.Store,
	destinationStore dstore.Store,
	resolver *accountsresolver.KVDBAccountsResolver,
	encoder firecore.BlockEncoder,
	logger *zap.Logger,
) error {

	paddedBlockNum := fmt.Sprintf("%010d", startBlock)
	logger.Info("Processing merge blocks",
		zap.Uint64("start_block_num", startBlock),
		zap.Uint64("stop_block_num", stopBlock),
		zap.String("first_merge_filename", paddedBlockNum),
	)

	mergeBlocksFileChan := make(chan *mergeBlocksFile, 20)
	done := make(chan interface{})

	go func() {
		err := processMergeBlocksFiles(ctx, mergeBlocksFileChan, destinationStore, resolver, encoder, logger)
		if err != nil {
			panic(fmt.Errorf("processing merge blocks files: %w", err))
		}
		close(done)
	}()

	err := sourceStore.WalkFrom(ctx, "", paddedBlockNum, func(filename string) error {
		mbf := newMergeBlocksFile(filename, logger)

		blkNumber, err := mbf.BlockNumber()
		if err != nil {
			return fmt.Errorf("converting block number of merged block file: %w", err)
		}
		if blkNumber >= stopBlock {
			logger.Info("Reached stop block", zap.Uint64("stop_block", stopBlock))
			close(mergeBlocksFileChan)
			return io.EOF
		}

		go func() {
			err := mbf.process(ctx, sourceStore)
			if err != nil {
				panic(fmt.Errorf("processing merge block file %s: %w", mbf.filename, err))
			}
		}()
		mergeBlocksFileChan <- mbf
		return nil
	})

	if err != nil && err != io.EOF {
		return fmt.Errorf("walking merge block sourceStore: %w", err)
	}

	logger.Info("Waiting for completion")
	<-done

	logger.Info("Done processing merge blocks")

	return nil
}

type bundleJob struct {
	filename     string
	bundleReader *accountsresolver.BundleReader
}

func processMergeBlocksFiles(
	ctx context.Context,
	mergeBlocksFileChan chan *mergeBlocksFile,
	destinationStore dstore.Store,
	resolver *accountsresolver.KVDBAccountsResolver,
	encoder firecore.BlockEncoder,
	logger *zap.Logger,
) error {

	writerNailer := dhammer.NewNailer(100, func(ctx context.Context, br *bundleJob) (*bundleJob, error) {
		logger.Info("nailing writing bundle file", zap.String("filename", br.filename))
		err := destinationStore.WriteObject(ctx, br.filename, br.bundleReader)
		if err != nil {
			return br, fmt.Errorf("writing bundle file: %w", err)
		}

		logger.Info("nailed writing bundle file", zap.String("filename", br.filename))
		return br, nil
	})
	writerNailer.OnTerminating(func(err error) {
		if err != nil {
			panic(fmt.Errorf("writing bundle file: %w", err))
		}
	})
	writerNailer.Start(ctx)
	done := make(chan interface{})

	go func() {
		for out := range writerNailer.Out {
			logger.Info("new merge blocks file written:", zap.String("filename", out.filename))
		}
		close(done)
	}()

	timeOfLastPush := time.Now()
	for mbf := range mergeBlocksFileChan {
		logger.Info("Receive merge block file", zap.String("filename", mbf.filename), zap.String("time_since_last push", durafmt.Parse(time.Since(timeOfLastPush)).String()))
		bundleReader := accountsresolver.NewBundleReader(ctx, logger)

		decoderNailer := dhammer.NewNailer(100, func(ctx context.Context, blk *pbsol.Block) (*bstream.Block, error) {
			b, err := encoder.Encode(blk)
			if err != nil {
				return nil, fmt.Errorf("encoding block: %w", err)
			}

			return b, nil
		})
		decoderNailer.OnTerminating(func(err error) {
			if err != nil {
				panic(fmt.Errorf("encoding block: %w", err))
			}
		})
		decoderNailer.Start(ctx)

		job := &bundleJob{
			mbf.filename,
			bundleReader,
		}
		writerNailer.Push(ctx, job)

		mbf := mbf
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case blk, ok := <-mbf.blockChan:
					if !ok {
						decoderNailer.Close()
						return
					}
					logger.Debug("handling block", zap.Uint64("slot", blk.Slot), zap.Uint64("parent_slot", blk.ParentSlot))

					err := processBlock(context.Background(), blk, resolver, logger)
					if err != nil {
						bundleReader.PushError(fmt.Errorf("processing block: %w", err))
						return
					}

					decoderNailer.Push(ctx, blk)
				}
			}
		}()
		for bb := range decoderNailer.Out {
			logger.Debug("pushing block", zap.Uint64("slot", bb.Num()))
			err := bundleReader.PushBlock(bb)
			if err != nil {
				bundleReader.PushError(fmt.Errorf("pushing block to bundle reader: %w", err))
				return fmt.Errorf("pushing block to bundle reader: %w", err)
			}
		}
		bundleReader.Close()
		timeOfLastPush = time.Now()
	}

	writerNailer.Close()
	logger.Info("Waiting for writer to complete")
	<-done
	logger.Info("Writer completed")

	return nil
}

func processBlock(ctx context.Context, block *pbsol.Block, resolver *accountsresolver.KVDBAccountsResolver, logger *zap.Logger) error {
	for _, trx := range block.Transactions {
		if trx.Meta.Err != nil {
			continue
		}
		//p.logger.Debug("processing transaction", zap.Uint64("block_num", block.Slot), zap.String("trx_id", base58.Encode(trx.Transaction.Signatures[0])))
		err := accountsresolver.ApplyTableLookup(ctx, accountsresolver.NewStats(), block.Slot, trx, resolver, logger)
		if err != nil {
			return fmt.Errorf("applying table lookup at block %d: %w", block.Slot, err)
		}
	}

	return nil
}

type mergeBlocksFile struct {
	filename  string
	blockChan chan *pbsol.Block
	logger    *zap.Logger
}

func newMergeBlocksFile(fileName string, logger *zap.Logger) *mergeBlocksFile {
	return &mergeBlocksFile{
		filename:  fileName,
		blockChan: make(chan *pbsol.Block, 100),
		logger:    logger,
	}
}

func (f *mergeBlocksFile) BlockNumber() (uint64, error) {
	return strconv.ParseUint(strings.TrimLeft(f.filename, "0"), 10, 64)
}

func (f *mergeBlocksFile) process(ctx context.Context, sourceStore dstore.Store) error {
	f.logger.Info("Processing merge block file", zap.String("filename", f.filename))
	firstBlockOfFile, err := strconv.Atoi(strings.TrimLeft(f.filename, "0"))
	if err != nil {
		return fmt.Errorf("converting filename to block number: %w", err)
	}

	reader, err := sourceStore.OpenObject(ctx, f.filename)
	if err != nil {
		return fmt.Errorf("opening merge block file %s: %w", f.filename, err)
	}
	defer reader.Close()

	blockReader, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		return fmt.Errorf("creating block reader for file %s: %w", f.filename, err)
	}

	for {
		block, err := blockReader.Read()
		if err != nil {
			if err == io.EOF {
				close(f.blockChan)
				return nil
			}
			return fmt.Errorf("reading block: %w", err)
		}

		blk := block.ToProtocol().(*pbsol.Block)
		if blk.Slot < uint64(firstBlockOfFile) {
			f.logger.Info("skip block process in previous file", zap.Uint64("slot", blk.Slot))
			continue
		}

		f.blockChan <- blk
	}
}
