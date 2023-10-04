package accountsresolver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/mr-tron/base58"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dhammer"
	"github.com/streamingfast/dstore"
	firecore "github.com/streamingfast/firehose-core"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/solana-go/programs/addresstablelookup"
	"go.uber.org/zap"
)

type Stats struct {
	startProcessing                    time.Time
	transactionCount                   int
	lookupCount                        int
	extendCount                        int
	totalBlockCount                    int
	totalLookupDuration                time.Duration
	totalTransactionProcessingDuration time.Duration
	totalExtendDuration                time.Duration
	totalBlockProcessingDuration       time.Duration
	totalBlockHandlingDuration         time.Duration
	totalBlockReadingDuration          time.Duration
	cacheHit                           int
	totalBlockPushDuration             time.Duration
	writeDurationAfterLastPush         time.Duration
	lastBlockPushedAt                  time.Time
}

func (s *Stats) Log(logger *zap.Logger) {
	lookupAvg := time.Duration(0)
	if s.lookupCount > 0 {
		lookupAvg = s.totalLookupDuration / time.Duration(s.lookupCount)
	}
	extendAvg := time.Duration(0)
	if s.extendCount > 0 {
		lookupAvg = s.totalExtendDuration / time.Duration(s.extendCount)
	}

	if s.totalBlockCount == 0 {
		logger.Info("no stats")
		return
	}

	logger.Info("stats",
		zap.Int("block_count", s.totalBlockCount),
		zap.Int("transaction_count", s.transactionCount),
		zap.Int("lookup_count", s.lookupCount),
		zap.Int("cache_hit", s.cacheHit),
		zap.Int("extend_count", s.extendCount),
		zap.String("total_block_handling_duration", durafmt.Parse(s.totalBlockHandlingDuration).String()),
		zap.String("total_block_processing_duration", durafmt.Parse(s.totalBlockProcessingDuration).String()),
		zap.String("total_transaction_processing_duration", durafmt.Parse(s.totalTransactionProcessingDuration).String()),
		zap.String("total_push_duration", durafmt.Parse(s.totalBlockPushDuration).String()),
		zap.String("total_lookup_duration", durafmt.Parse(s.totalLookupDuration).String()),
		zap.String("total_extend_duration", durafmt.Parse(s.totalExtendDuration).String()),
		zap.String("total_duration", durafmt.Parse(time.Since(s.startProcessing)).String()),
		zap.String("total_block_reading_duration", durafmt.Parse(s.totalBlockReadingDuration).String()),
		zap.String("average_block_handling_duration", durafmt.Parse(s.totalBlockHandlingDuration/time.Duration(s.totalBlockCount)).String()),
		zap.String("average_block_processing_duration", durafmt.Parse(s.totalBlockProcessingDuration/time.Duration(s.totalBlockCount)).String()),
		zap.String("average_transaction_processing_duration", durafmt.Parse(s.totalTransactionProcessingDuration/time.Duration(s.transactionCount)).String()),
		zap.String("average_lookup_duration", durafmt.Parse(lookupAvg).String()),
		zap.String("average_extend_duration", durafmt.Parse(extendAvg).String()),
		zap.String("write_duration_after_last_push", durafmt.Parse(time.Since(s.lastBlockPushedAt)).String()),
	)
}

var AddressTableLookupAccountProgram = mustFromBase58("AddressLookupTab1e1111111111111111111111111")
var SystemProgram = mustFromBase58("11111111111111111111111111111111")

type Cursor struct {
	slotNum uint64
}

func NewCursor(blockNum uint64) *Cursor {
	return &Cursor{
		slotNum: blockNum,
	}
}

type Processor struct {
	accountsResolver AccountsResolver
	readerName       string
	logger           *zap.Logger
}

func NewProcessor(readerName string, accountsResolver AccountsResolver, logger *zap.Logger) *Processor {
	return &Processor{
		readerName:       readerName,
		accountsResolver: accountsResolver,
		logger:           logger,
	}
}

func (p *Processor) ProcessMergeBlocks(ctx context.Context, cursor *Cursor, sourceStore dstore.Store, destinationStore dstore.Store, encoder firecore.BlockEncoder) error {
	startBlockNum := cursor.slotNum - cursor.slotNum%100 //This is the first block slot of the last merge block file
	startBlockNum += 100                                 //This is the first block slot of the next merge block file
	paddedBlockNum := fmt.Sprintf("%010d", startBlockNum)

	p.logger.Info("Processing merge blocks", zap.Uint64("cursor_block_num", cursor.slotNum), zap.String("first_merge_filename", paddedBlockNum))

	mergeBlocksFileChan := make(chan *mergeBlocksFile, 20)

	go func() {
		err := p.processMergeBlocksFiles(ctx, cursor, mergeBlocksFileChan, destinationStore, encoder)
		panic(fmt.Errorf("processing merge blocks files: %w", err))
	}()

	err := sourceStore.WalkFrom(ctx, "", paddedBlockNum, func(filename string) error {
		mbf := newMergeBlocksFile(filename, p.logger)
		go func() {
			err := mbf.process(ctx, sourceStore)
			if err != nil {
				panic(fmt.Errorf("processing merge block file %s: %w", mbf.filename, err))
			}
		}()
		mergeBlocksFileChan <- mbf
		return nil
	})

	if err != nil {
		return fmt.Errorf("walking merge block sourceStore: %w", err)
	}

	p.logger.Info("Done processing merge blocks")

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
		blockChan: make(chan *pbsol.Block, 1),
		logger:    logger,
	}
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

type bundleJob struct {
	filename     string
	cursor       *Cursor
	stats        *Stats
	bundleReader *BundleReader
}

func (p *Processor) processMergeBlocksFiles(ctx context.Context, cursor *Cursor, mergeBlocksFileChan chan *mergeBlocksFile, destinationStore dstore.Store, encoder firecore.BlockEncoder) error {

	writerNailer := dhammer.NewNailer(50, func(ctx context.Context, br *bundleJob) (*bundleJob, error) {
		err := destinationStore.WriteObject(ctx, br.filename, br.bundleReader)
		if err != nil {
			return br, fmt.Errorf("writing bundle file: %w", err)
		}

		return br, nil
	})
	writerNailer.OnTerminating(func(err error) {
		if err != nil {
			panic(fmt.Errorf("writing bundle file: %w", err))
		}
	})
	writerNailer.Start(ctx)

	go func() {
		for out := range writerNailer.Out {
			p.logger.Info("new merge blocks file written:", zap.String("filename", out.filename))
			err := p.accountsResolver.StoreCursor(ctx, p.readerName, out.cursor)
			if err != nil {
				panic(fmt.Errorf("storing cursor at block %d: %w", out.cursor.slotNum, err))
			}
			out.stats.Log(p.logger)
			out.bundleReader.Close()
		}
	}()

	for mbf := range mergeBlocksFileChan {
		stats := &Stats{
			startProcessing: time.Now(),
		}
		p.logger.Info("Receive merge block file", zap.String("filename", mbf.filename))
		bundleReader := NewBundleReader(ctx, p.logger)

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

		writerNailer.Push(ctx, &bundleJob{
			mbf.filename,
			cursor,
			stats,
			bundleReader,
		})

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

					if blk.Slot <= cursor.slotNum {
						p.logger.Info("skip block", zap.Uint64("slot", blk.Slot))
						continue
					}

					if cursor.slotNum != blk.ParentSlot {
						bundleReader.PushError(fmt.Errorf("cursor block num %d is not the same as parent slot num %d of block %d", cursor.slotNum, blk.ParentSlot, blk.Slot))
						return
					}

					start := time.Now()
					err := p.ProcessBlock(context.Background(), stats, blk)
					if err != nil {
						bundleReader.PushError(fmt.Errorf("processing block: %w", err))
						return
					}

					stats.totalBlockProcessingDuration += time.Since(start)

					cursor.slotNum = blk.Slot
					decoderNailer.Push(ctx, blk)

					stats.totalBlockHandlingDuration += time.Since(start)
				}
			}
		}()
		for bb := range decoderNailer.Out {
			err := bundleReader.PushBlock(bb)
			pushStart := time.Now()
			if err != nil {
				bundleReader.PushError(fmt.Errorf("pushing block to bundle reader: %w", err))
				return fmt.Errorf("pushing block to bundle reader: %w", err)
			}
			stats.totalBlockPushDuration += time.Since(pushStart)
		}
		stats.lastBlockPushedAt = time.Now()
	}

	return nil
}

func (p *Processor) ProcessBlock(ctx context.Context, stats *Stats, block *pbsol.Block) error {
	stats.transactionCount += len(block.Transactions)
	for _, trx := range block.Transactions {
		if trx.Meta.Err != nil {
			continue
		}
		//p.logger.Debug("processing transaction", zap.Uint64("block_num", block.Slot), zap.String("trx_id", base58.Encode(trx.Transaction.Signatures[0])))
		err := p.applyTableLookup(ctx, stats, block.Slot, trx)
		if err != nil {
			return fmt.Errorf("applying table lookup at block %d: %w", block.Slot, err)
		}

		err = p.manageAddressLookup(ctx, stats, block.Slot, err, trx)
		if err != nil {
			return fmt.Errorf("managing address lookup at block %d: %w", block.Slot, err)
		}
	}
	stats.totalBlockCount += 1

	return nil
}

func (p *Processor) manageAddressLookup(ctx context.Context, stats *Stats, blockNum uint64, err error, trx *pbsol.ConfirmedTransaction) error {
	err = p.ProcessTransaction(ctx, stats, blockNum, trx)
	if err != nil {
		return fmt.Errorf("processing transactions: %w", err)
	}
	return nil
}

func (p *Processor) applyTableLookup(ctx context.Context, stats *Stats, blockNum uint64, trx *pbsol.ConfirmedTransaction) error {
	start := time.Now()
	for _, addressTableLookup := range trx.Transaction.Message.AddressTableLookups {
		accs, cached, err := p.accountsResolver.Resolve(ctx, blockNum, addressTableLookup.AccountKey)
		if err != nil {
			return fmt.Errorf("resolving address table %s at block %d: %w", base58.Encode(addressTableLookup.AccountKey), blockNum, err)
		}
		if cached {
			stats.cacheHit += 1
		}
		//p.logger.Debug("Resolve address table lookup", zap.String("trx", getTransactionHash(trx.Transaction.Signatures)), zap.String("account", base58.Encode(addressTableLookup.AccountKey)), zap.Int("count", len(accs)), zap.Bool("cached", cached))
		trx.Transaction.Message.AccountKeys = append(trx.Transaction.Message.AccountKeys, accs.ToBytesArray()...)
	}
	totalDuration := time.Since(start)
	lookupCount := len(trx.Transaction.Message.AddressTableLookups)

	if lookupCount > 0 {
		stats.lookupCount += lookupCount
		stats.totalLookupDuration += totalDuration
		p.logger.Debug(
			"applyTableLookup",
			zap.Duration("duration", totalDuration),
			zap.Int("lookup_count", lookupCount),
			zap.Int64("average_lookup_time", totalDuration.Milliseconds()/int64(lookupCount)),
		)

	}
	return nil
}

func (p *Processor) ProcessTransaction(ctx context.Context, stats *Stats, blockNum uint64, confirmedTransaction *pbsol.ConfirmedTransaction) error {
	start := time.Now()
	accountKeys := confirmedTransaction.Transaction.Message.AccountKeys
	for compileIndex, compiledInstruction := range confirmedTransaction.Transaction.Message.Instructions {
		idx := compiledInstruction.ProgramIdIndex
		err := p.ProcessInstruction(ctx, stats, blockNum, confirmedTransaction.Transaction.Signatures[0], confirmedTransaction.Transaction.Message.AccountKeys[idx], accountKeys, compiledInstruction)
		if err != nil {
			return fmt.Errorf("confirmedTransaction %s processing compiled instruction: %w", getTransactionHash(confirmedTransaction.Transaction.Signatures), err)
		}
		//todo; only inner instructions of compiled instructions
		if compileIndex+1 > len(confirmedTransaction.Meta.InnerInstructions) {
			continue
		}
		inner := confirmedTransaction.Meta.InnerInstructions[compileIndex]
		for _, instruction := range inner.Instructions {
			if len(accountKeys) < int(instruction.ProgramIdIndex) {
				return fmt.Errorf("missing account key at index %d for transaction %s with account keys count of %d", instruction.ProgramIdIndex, getTransactionHash(confirmedTransaction.Transaction.Signatures), len(accountKeys))
			}

			err := p.ProcessInstruction(ctx, stats, blockNum, confirmedTransaction.Transaction.Signatures[0], accountKeys[instruction.ProgramIdIndex], accountKeys, instruction)
			if err != nil {
				return fmt.Errorf("confirmedTransaction %s processing instruxction: %w", getTransactionHash(confirmedTransaction.Transaction.Signatures), err)
			}
		}
	}
	stats.totalTransactionProcessingDuration += time.Since(start)
	return nil
}

func (p *Processor) ProcessInstruction(ctx context.Context, stats *Stats, blockNum uint64, trxHash []byte, programAccount Account, accountKeys [][]byte, instructionable pbsol.Instructionable) error {
	if !bytes.Equal(programAccount, AddressTableLookupAccountProgram) {
		return nil
	}

	instruction := instructionable.ToInstruction()
	if addresstablelookup.ExtendAddressTableLookupInstruction(instruction.Data) {
		start := time.Now()

		tableLookupAccount := accountKeys[instruction.Accounts[0]]
		newAccounts := addresstablelookup.ParseNewAccounts(instruction.Data[12:])
		p.logger.Debug("Extending address table lookup", zap.String("account", base58.Encode(tableLookupAccount)), zap.Int("new_account_count", len(newAccounts)))
		err := p.accountsResolver.Extend(ctx, blockNum, trxHash, tableLookupAccount, NewAccounts(newAccounts))

		if err != nil {
			return fmt.Errorf("extending address table %s at block %d: %w", tableLookupAccount, blockNum, err)
		}

		stats.totalExtendDuration += time.Since(start)
		stats.extendCount += 1
	}

	return nil
}

func getTransactionHash(signatures [][]byte) string {
	return base58.Encode(signatures[0])
}
