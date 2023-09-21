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

type stats struct {
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
}

func (s *stats) log(logger *zap.Logger) {
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
		zap.String("total_lookup_duration", durafmt.Parse(s.totalLookupDuration).String()),
		zap.String("total_extend_duration", durafmt.Parse(s.totalExtendDuration).String()),
		zap.String("total_duration", durafmt.Parse(time.Since(s.startProcessing)).String()),
		zap.String("total_block_reading_duration", durafmt.Parse(s.totalBlockReadingDuration).String()),
		zap.String("average_block_handling_duration", durafmt.Parse(s.totalBlockHandlingDuration/time.Duration(s.totalBlockCount)).String()),
		zap.String("average_block_processing_duration", durafmt.Parse(s.totalBlockProcessingDuration/time.Duration(s.totalBlockCount)).String()),
		zap.String("average_transaction_processing_duration", durafmt.Parse(s.totalTransactionProcessingDuration/time.Duration(s.transactionCount)).String()),
		zap.String("average_lookup_duration", durafmt.Parse(lookupAvg).String()),
		zap.String("average_extend_duration", durafmt.Parse(extendAvg).String()),
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
	cursor           *Cursor
	readerName       string
	logger           *zap.Logger
	stats            *stats
}

func NewProcessor(readerName string, cursor *Cursor, accountsResolver AccountsResolver, logger *zap.Logger) *Processor {
	return &Processor{
		readerName:       readerName,
		accountsResolver: accountsResolver,
		cursor:           cursor,
		logger:           logger,
		stats:            &stats{},
	}
}

func (p *Processor) ProcessMergeBlocks(ctx context.Context, sourceStore dstore.Store, destinationStore dstore.Store, encoder firecore.BlockEncoder) error {
	startBlockNum := p.cursor.slotNum - p.cursor.slotNum%100
	paddedBlockNum := fmt.Sprintf("%010d", startBlockNum)

	p.logger.Info("Processing merge blocks", zap.Uint64("cursor_block_num", p.cursor.slotNum), zap.String("first_merge_filename", paddedBlockNum))

	err := sourceStore.WalkFrom(ctx, "", paddedBlockNum, func(filename string) error {
		p.logger.Debug("processing merge block file", zap.String("filename", filename))
		return p.processMergeBlocksFiles(ctx, filename, sourceStore, destinationStore, encoder)
	})

	if err != nil {
		return fmt.Errorf("walking merge block sourceStore: %w", err)
	}

	p.logger.Info("Done processing merge blocks")

	return nil
}

func (p *Processor) processMergeBlocksFiles(ctx context.Context, filename string, sourceStore dstore.Store, destinationStore dstore.Store, encoder firecore.BlockEncoder) error {
	p.logger.Info("Processing merge block file", zap.String("filename", filename))
	p.stats = &stats{
		startProcessing: time.Now(),
	}

	firstBlockOfFile, err := strconv.Atoi(strings.TrimLeft(filename, "0"))
	if err != nil {
		return fmt.Errorf("converting filename to block number: %w", err)
	}

	reader, err := sourceStore.OpenObject(ctx, filename)
	if err != nil {
		return fmt.Errorf("opening merge block file %s: %w", filename, err)
	}
	defer reader.Close()

	blockReader, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		return fmt.Errorf("creating block reader for file %s: %w", filename, err)
	}

	bundleReader := NewBundleReader(ctx, p.logger)
	blockChan := make(chan *pbsol.Block, 100)

	go func() {
		start := time.Now()
		for {
			block, err := blockReader.Read()
			if err != nil {
				if err == io.EOF {
					close(blockChan)
					return
				}
				bundleReader.PushError(fmt.Errorf("reading block: %w", err))
				return
			}

			blk := block.ToProtocol().(*pbsol.Block)
			if blk.Slot < uint64(firstBlockOfFile) || blk.Slot <= p.cursor.slotNum {
				p.logger.Debug("skip block", zap.Uint64("slot", blk.Slot))
				continue
			}

			blockChan <- blk
		}
		p.stats.totalBlockReadingDuration += time.Since(start)
	}()

	nailer := dhammer.NewNailer(50, func(ctx context.Context, blk *pbsol.Block) (*bstream.Block, error) {
		b, err := encoder.Encode(blk)
		if err != nil {
			return nil, fmt.Errorf("encoding block: %w", err)
		}

		return b, nil
	})
	nailer.Start(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case blk, ok := <-blockChan:
				if !ok {
					nailer.Close()
					return
				}

				start := time.Now()
				err := p.ProcessBlock(context.Background(), blk)
				if err != nil {
					bundleReader.PushError(fmt.Errorf("processing block: %w", err))
					return
				}
				p.stats.totalBlockProcessingDuration += time.Since(start)

				nailer.Push(ctx, blk)
				p.stats.totalBlockCount += 1
				p.stats.totalBlockHandlingDuration += time.Since(start)
			}
		}
	}()

	go func() {
		for bb := range nailer.Out {
			err = bundleReader.PushBlock(bb)
			if err != nil {
				bundleReader.PushError(fmt.Errorf("pushing block to bundle reader: %w", err))
				return
			}
		}
		bundleReader.Close()
	}()

	err = destinationStore.WriteObject(ctx, filename, bundleReader)
	if err != nil {
		return fmt.Errorf("writing bundle file: %w", err)
	}
	//p.logger.Info("new merge blocks file written:", zap.String("filename", filename), zap.Duration("duration", time.Since(start)))
	err = p.accountsResolver.StoreCursor(ctx, p.readerName, p.cursor)
	if err != nil {
		return fmt.Errorf("storing cursor at block %d: %w", p.cursor.slotNum, err)
	}

	p.stats.log(p.logger)
	return nil
}

func (p *Processor) ProcessBlock(ctx context.Context, block *pbsol.Block) error {
	if p.cursor == nil {
		return fmt.Errorf("cursor is nil")
	}

	if p.cursor.slotNum != block.ParentSlot {
		return fmt.Errorf("cursor block num %d is not the same as parent slot num %d of block %d", p.cursor.slotNum, block.ParentSlot, block.Slot)
	}
	p.stats.transactionCount += len(block.Transactions)
	for _, trx := range block.Transactions {
		if trx.Meta.Err != nil {
			continue
		}
		//p.logger.Debug("processing transaction", zap.Uint64("block_num", block.Slot), zap.String("trx_id", base58.Encode(trx.Transaction.Signatures[0])))
		err := p.applyTableLookup(ctx, block.Slot, trx)
		if err != nil {
			return fmt.Errorf("applying table lookup at block %d: %w", block.Slot, err)
		}

		err = p.manageAddressLookup(ctx, block.Slot, err, trx)
		if err != nil {
			return fmt.Errorf("managing address lookup at block %d: %w", block.Slot, err)
		}
	}

	p.cursor.slotNum = block.Slot

	return nil
}

func (p *Processor) manageAddressLookup(ctx context.Context, blockNum uint64, err error, trx *pbsol.ConfirmedTransaction) error {
	err = p.ProcessTransaction(ctx, blockNum, trx)
	if err != nil {
		return fmt.Errorf("processing transactions: %w", err)
	}
	return nil
}

func (p *Processor) applyTableLookup(ctx context.Context, blockNum uint64, trx *pbsol.ConfirmedTransaction) error {
	start := time.Now()
	for _, addressTableLookup := range trx.Transaction.Message.AddressTableLookups {
		accs, cached, err := p.accountsResolver.Resolve(ctx, blockNum, addressTableLookup.AccountKey)
		if err != nil {
			return fmt.Errorf("resolving address table %s at block %d: %w", base58.Encode(addressTableLookup.AccountKey), blockNum, err)
		}
		if cached {
			p.stats.cacheHit += 1
		}
		p.logger.Debug("Resolve address table lookup", zap.String("trx", getTransactionHash(trx.Transaction.Signatures)), zap.String("account", base58.Encode(addressTableLookup.AccountKey)), zap.Int("count", len(accs)))
		trx.Transaction.Message.AccountKeys = append(trx.Transaction.Message.AccountKeys, accs.ToBytesArray()...)
	}
	totalDuration := time.Since(start)
	lookupCount := len(trx.Transaction.Message.AddressTableLookups)

	if lookupCount > 0 {
		p.stats.lookupCount += lookupCount
		p.stats.totalLookupDuration += totalDuration
		p.logger.Debug(
			"applyTableLookup",
			zap.Duration("duration", totalDuration),
			zap.Int("lookup_count", lookupCount),
			zap.Int64("average_lookup_time", totalDuration.Milliseconds()/int64(lookupCount)),
		)

	}
	return nil
}

func (p *Processor) ProcessTransaction(ctx context.Context, blockNum uint64, confirmedTransaction *pbsol.ConfirmedTransaction) error {
	start := time.Now()
	accountKeys := confirmedTransaction.Transaction.Message.AccountKeys
	for compileIndex, compiledInstruction := range confirmedTransaction.Transaction.Message.Instructions {
		idx := compiledInstruction.ProgramIdIndex
		err := p.ProcessInstruction(ctx, blockNum, confirmedTransaction.Transaction.Signatures[0], confirmedTransaction.Transaction.Message.AccountKeys[idx], accountKeys, compiledInstruction)
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

			err := p.ProcessInstruction(ctx, blockNum, confirmedTransaction.Transaction.Signatures[0], accountKeys[instruction.ProgramIdIndex], accountKeys, instruction)
			if err != nil {
				return fmt.Errorf("confirmedTransaction %s processing instruxction: %w", getTransactionHash(confirmedTransaction.Transaction.Signatures), err)
			}
		}
	}
	p.stats.totalTransactionProcessingDuration += time.Since(start)
	return nil
}

func (p *Processor) ProcessInstruction(ctx context.Context, blockNum uint64, trxHash []byte, programAccount Account, accountKeys [][]byte, instructionable pbsol.Instructionable) error {
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

		p.stats.totalExtendDuration += time.Since(start)
		p.stats.extendCount += 1
	}

	return nil
}

func getTransactionHash(signatures [][]byte) string {
	return base58.Encode(signatures[0])
}
