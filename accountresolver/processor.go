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
	"github.com/streamingfast/dstore"
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
	totalDecodingDuration              time.Duration
	timeToFirstDecodedBlock            time.Duration
	totalTimeWaitingForBlock           time.Duration
	totalAccountsResolved              int
	totalAccountsResolvedByCache       int
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
		zap.Int("total_accounts_resolved", s.totalAccountsResolved),
		zap.Int("total_accounts_resolved_by_cache", s.totalAccountsResolvedByCache),
		zap.String("total_block_handling_duration", durafmt.Parse(s.totalBlockHandlingDuration).String()),
		zap.String("total_block_processing_duration", durafmt.Parse(s.totalBlockProcessingDuration).String()),
		zap.String("total_transaction_processing_duration", durafmt.Parse(s.totalTransactionProcessingDuration).String()),
		zap.String("total_push_duration", durafmt.Parse(s.totalBlockPushDuration).String()),
		zap.String("total_lookup_duration", durafmt.Parse(s.totalLookupDuration).String()),
		zap.String("total_extend_duration", durafmt.Parse(s.totalExtendDuration).String()),
		zap.String("total_duration", durafmt.Parse(time.Since(s.startProcessing)).String()),
		zap.String("total_block_reading_duration", durafmt.Parse(s.totalBlockReadingDuration).String()),
		zap.String("total_decoding_duration", durafmt.Parse(s.totalDecodingDuration).String()),
		zap.String("total_time_waiting_for_block", durafmt.Parse(s.totalTimeWaitingForBlock).String()),
		zap.String("total_time_waiting_for_block", durafmt.Parse(s.totalTimeWaitingForBlock).String()),
		//zap.String("average_block_handling_duration", durafmt.Parse(s.totalBlockHandlingDuration/time.Duration(s.totalBlockCount)).String()),
		//zap.String("average_block_processing_duration", durafmt.Parse(s.totalBlockProcessingDuration/time.Duration(s.totalBlockCount)).String()),
		//zap.String("average_transaction_processing_duration", durafmt.Parse(s.totalTransactionProcessingDuration/time.Duration(s.transactionCount)).String()),
		zap.String("average_lookup_duration", durafmt.Parse(lookupAvg).String()),
		zap.String("average_extend_duration", durafmt.Parse(extendAvg).String()),
		zap.String("write_duration_after_last_push", durafmt.Parse(time.Since(s.lastBlockPushedAt)).String()),
		zap.String("time_to_first_decoded_block", durafmt.Parse(s.timeToFirstDecodedBlock).String()),
	)
}

var AddressTableLookupAccountProgram = MustFromBase58("AddressLookupTab1e1111111111111111111111111")
var SystemProgram = MustFromBase58("11111111111111111111111111111111")

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

func (p *Processor) ProcessMergeBlocks(ctx context.Context, cursor *Cursor, sourceStore dstore.Store) error {
	startBlockNum := cursor.slotNum - cursor.slotNum%100 //This is the first block slot of the last merge block file
	startBlockNum += 100                                 //This is the first block slot of the next merge block file
	paddedBlockNum := fmt.Sprintf("%010d", startBlockNum)

	p.logger.Info("Processing merge blocks", zap.Uint64("cursor_block_num", cursor.slotNum), zap.String("first_merge_filename", paddedBlockNum))

	downloadedMergeBlocksFileChan := make(chan *mergeBlocksFile, 20)

	go func() {
		err := p.processMergeBlocksFiles(ctx, cursor, downloadedMergeBlocksFileChan)
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
		downloadedMergeBlocksFileChan <- mbf
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
		blockChan: make(chan *pbsol.Block, 100),
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

func (p *Processor) processMergeBlocksFiles(ctx context.Context, cursor *Cursor, mergeBlocksFileChan chan *mergeBlocksFile) error {
	timeOfLastPush := time.Now()
	for mbf := range mergeBlocksFileChan {
		p.logger.Info("Receive merge block file", zap.String("filename", mbf.filename), zap.String("time_since_last_process_", durafmt.Parse(time.Since(timeOfLastPush)).String()))
		stats := &Stats{
			startProcessing: time.Now(),
		}

		for blk := range mbf.blockChan {
			select {
			case <-ctx.Done():
				return nil
			default:
				startWaiting := time.Now()
				stats.totalTimeWaitingForBlock += time.Since(startWaiting)
				if blk.Slot <= cursor.slotNum {
					p.logger.Info("skip block", zap.Uint64("slot", blk.Slot))
					continue
				}
				p.logger.Debug("handling block", zap.Uint64("slot", blk.Slot), zap.Uint64("parent_slot", blk.ParentSlot))
				if cursor.slotNum != blk.ParentSlot {
					return fmt.Errorf("cursor block num %d is not the same as parent slot num %d of block %d", cursor.slotNum, blk.ParentSlot, blk.Slot)
				}

				start := time.Now()
				err := p.ProcessBlock(context.Background(), stats, blk)
				if err != nil {
					return fmt.Errorf("processing block: %w", err)
				}

				stats.totalBlockProcessingDuration += time.Since(start)

				cursor.slotNum = blk.Slot
				stats.totalBlockHandlingDuration += time.Since(start)
			}
		}
		err := p.accountsResolver.StoreCursor(ctx, p.readerName, cursor)
		if err != nil {
			panic(fmt.Errorf("storing cursor at block %d: %w", cursor.slotNum, err))
		}
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
		resolvedAccounts, cached, err := p.accountsResolver.Resolve(ctx, blockNum, addressTableLookup.AccountKey)
		if err != nil {
			return fmt.Errorf("resolving address table %s at block %d: %w", base58.Encode(addressTableLookup.AccountKey), blockNum, err)
		}

		if len(resolvedAccounts) == 0 {
			p.logger.Warn("Resolved accounts is empty", zap.Uint64("block", blockNum), zap.String("table account", base58.Encode(addressTableLookup.AccountKey)), zap.Bool("cached", cached), zap.Int("account_count", len(resolvedAccounts)))
		}

		if cached {
			stats.cacheHit += 1
			stats.totalAccountsResolvedByCache += len(resolvedAccounts)
		}
		stats.totalAccountsResolved += len(resolvedAccounts)

		//p.logger.Info("resolved accounts", zap.Uint64("block", blockNum), zap.String("table account", base58.Encode(addressTableLookup.AccountKey)), zap.Int("account_count", len(resolvedAccounts)))

		for _, index := range addressTableLookup.WritableIndexes {
			if int(index) >= len(resolvedAccounts) {
				return fmt.Errorf("missing writable account key from %s at index %d for transaction %s with account keys count of %d at block %d cached: %t", base58.Encode(addressTableLookup.AccountKey), index, getTransactionHash(trx.Transaction.Signatures), len(resolvedAccounts), blockNum, cached)
			}
			trx.Transaction.Message.AccountKeys = append(trx.Transaction.Message.AccountKeys, resolvedAccounts[index])
		}

		for _, index := range addressTableLookup.ReadonlyIndexes {
			if int(index) >= len(resolvedAccounts) {
				return fmt.Errorf("missing readonly account key from %s at index %d for transaction %s with account keys count of %d at block %d cached: %t", base58.Encode(addressTableLookup.AccountKey), index, getTransactionHash(trx.Transaction.Signatures), len(resolvedAccounts), blockNum, cached)
			}
			trx.Transaction.Message.AccountKeys = append(trx.Transaction.Message.AccountKeys, resolvedAccounts[index])
		}
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
	if confirmedTransaction.Meta.Err != nil {
		p.logger.Info("skipping transaction with error", zap.Uint64("block_num", blockNum), zap.String("trx_id", base58.Encode(confirmedTransaction.Transaction.Signatures[0])))
		return nil
	}
	accountKeys := confirmedTransaction.Transaction.Message.AccountKeys
	for instructionIndex, compiledInstruction := range confirmedTransaction.Transaction.Message.Instructions {
		idx := compiledInstruction.ProgramIdIndex
		err := p.ProcessInstruction(ctx, stats, blockNum, confirmedTransaction.Transaction.Signatures[0], fmt.Sprintf("%d", instructionIndex), confirmedTransaction.Transaction.Message.AccountKeys[idx], accountKeys, compiledInstruction)
		if err != nil {
			return fmt.Errorf("confirmedTransaction %s processing compiled instruction: %w", getTransactionHash(confirmedTransaction.Transaction.Signatures), err)
		}
		inner := GetInnerInstructions(instructionIndex, confirmedTransaction.Meta.InnerInstructions)
		if inner == nil {
			continue // there are no inner instructions for the CompiledInstruction
		}
		for i, instruction := range inner.Instructions {
			index := fmt.Sprintf("%d.%d", instructionIndex, i)
			if len(accountKeys) < int(instruction.ProgramIdIndex) {
				return fmt.Errorf("missing account key at instructionIndex %d for transaction %s with account keys count of %d", instruction.ProgramIdIndex, getTransactionHash(confirmedTransaction.Transaction.Signatures), len(accountKeys))
			}

			err := p.ProcessInstruction(ctx, stats, blockNum, confirmedTransaction.Transaction.Signatures[0], index, accountKeys[instruction.ProgramIdIndex], accountKeys, instruction)
			if err != nil {
				return fmt.Errorf("confirmedTransaction %s processing instruxction: %w", getTransactionHash(confirmedTransaction.Transaction.Signatures), err)
			}
		}
	}
	stats.totalTransactionProcessingDuration += time.Since(start)
	return nil
}

func GetInnerInstructions(index int, trxMetaInnerInstructions []*pbsol.InnerInstructions) *pbsol.InnerInstructions {
	for _, innerInstructions := range trxMetaInnerInstructions {
		if int(innerInstructions.Index) == index {
			return innerInstructions
		}
	}
	return nil
}

func (p *Processor) ProcessInstruction(ctx context.Context, stats *Stats, blockNum uint64, trxHash []byte, instructionIndex string, programAccount Account, accountKeys [][]byte, instructionable pbsol.Instructionable) error {
	if !bytes.Equal(programAccount, AddressTableLookupAccountProgram) {
		return nil
	}

	instruction := instructionable.ToInstruction()
	decodedInstruction, err := addresstablelookup.DecodeInstruction(instruction.Data)
	if err != nil {
		return fmt.Errorf("decoding instruction: %w", err)
	}

	switch val := decodedInstruction.Impl.(type) {
	case *addresstablelookup.ExtendLookupTable:
		start := time.Now()
		tableLookupAccount := accountKeys[instruction.Accounts[0]]
		newAccounts := make([][]byte, len(val.Addresses))
		for i := range val.Addresses {
			newAccounts[i] = val.Addresses[i][:]
		}
		p.logger.Debug("Extending address table lookup", zap.String("account", base58.Encode(tableLookupAccount)), zap.Int("new_account_count", len(newAccounts)))
		err := p.accountsResolver.Extend(ctx, blockNum, trxHash, instructionIndex, tableLookupAccount, NewAccounts(newAccounts))
		if err != nil {
			return fmt.Errorf("extending address table %s at block %d: %w", tableLookupAccount, blockNum, err)
		}

		stats.totalExtendDuration += time.Since(start)
		stats.extendCount += 1

	default:
		// only interested in extend lookup table instruction
	}

	return nil
}

func getTransactionHash(signatures [][]byte) string {
	return base58.Encode(signatures[0])
}
