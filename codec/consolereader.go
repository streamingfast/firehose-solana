// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codec

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"go.uber.org/zap"
)

var MaxTokenSize uint64

func init() {
	MaxTokenSize = uint64(50 * 1024 * 1024)
	if maxBufferSize := os.Getenv("MINDREADER_MAX_TOKEN_SIZE"); maxBufferSize != "" {
		bs, err := strconv.ParseUint(maxBufferSize, 10, 64)
		if err != nil {
			zlog.Error("environment variable 'MINDREADER_MAX_TOKEN_SIZE' is set but invalid parse uint", zap.Error(err))
		} else {
			zlog.Info("setting max_token_size from environment variable MINDREADER_MAX_TOKEN_SIZE", zap.Uint64("max_token_size", bs))
			MaxTokenSize = bs
		}
	}
}

var supportedVersions = []string{"1", "1"}

type conversionOption interface{}

type ConsoleReaderOption interface {
	apply(reader *ConsoleReader)
}

// ConsoleReader is what reads the `nodeos` output directly. It builds
// up some LogEntry objects. See `LogReader to read those entries .
type ConsoleReader struct {
	lines chan string
	close func()

	done           chan interface{}
	ctx            *parseCtx
	batchFilesPath string
}

func (r *ConsoleReader) ProcessData(reader io.Reader) error {
	scanner := r.buildScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		r.lines <- line
	}

	if scanner.Err() == nil {
		close(r.lines)
		return io.EOF
	}

	return scanner.Err()
}

func (r *ConsoleReader) buildScanner(reader io.Reader) *bufio.Scanner {
	buf := make([]byte, 50*1024*1024)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(buf, 50*1024*1024)

	return scanner
}

func NewConsoleReader(lines chan string, batchFilesPath string, opts ...ConsoleReaderOption) (*ConsoleReader, error) {
	l := &ConsoleReader{
		lines: lines,
		close: func() {},
		ctx:   newParseCtx(batchFilesPath),
		done:  make(chan interface{}),
	}

	for _, opt := range opts {
		opt.apply(l)
	}

	return l, nil
}

func (r *ConsoleReader) Done() <-chan interface{} {
	return r.done
}

func (r *ConsoleReader) Close() {
	r.close()
}

type parseCtx struct {
	activeBank        *bank
	banks             map[uint64]*bank
	conversionOptions []conversionOption
	blockBuffer       chan *pbcodec.Block
	batchWG           sync.WaitGroup
	batchFilesPath    string
}

func newParseCtx(batchFilesPath string) *parseCtx {
	return &parseCtx{
		banks:          map[uint64]*bank{},
		blockBuffer:    make(chan *pbcodec.Block, 10000),
		batchFilesPath: batchFilesPath,
	}
}

func (r *ConsoleReader) Read() (out interface{}, err error) {
	return r.next()
}

func (r *ConsoleReader) next() (out interface{}, err error) {
	ctx := r.ctx

	select {
	case s := <-ctx.blockBuffer:
		return s, nil
	default:
	}

	for line := range r.lines {
		fmt.Println("processing lines")
		if !strings.HasPrefix(line, "DMLOG ") {
			continue
		}

		line = line[6:] // removes the DMLOG prefix
		if err = parseLine(ctx, line); err != nil {
			return nil, r.formatError(line, err)
		}

		select {
		case s := <-ctx.blockBuffer:
			return s, nil
		default:
		}
	}

	zlog.Info("lines channel has been closed")
	return nil, io.EOF
}

func parseLine(ctx *parseCtx, line string) (err error) {
	// Order of conditions is based (approximately) on those that will appear more often
	switch {
	// defines the current version of deepmind; should fail is the value is unexpected
	case strings.HasPrefix(line, "INIT"):
		err = ctx.readInit(line)

	// this occurs at the beginning execution of a given block (bank) (this is a 'range' of slot say from 10 to 13,
	// it can also just be one slot), this can be PARTIAL or FULL work of said block. A given block may have multiple
	// SLOT_WORK partial but only one SLOT_WORK full.
	case strings.HasPrefix(line, "BLOCK_WORK"):
		err = ctx.readBlockWork(line)

	// output when a group of batch of transaction have been executed and the protobuf has been written to a file on  disk
	case strings.HasPrefix(line, "BATCH_FILE"):
		err = ctx.readBatchFile(line)

	// When executing a transactions, we will group them in multiples batches and run them in parallel.
	// We will create one file per batch (group of trxs), each batch is is running in its own thread.
	// When a given batch is completed we will receive BATCH_FILE. Once all the batches are completed in parallel
	// we will receive BATCH_END. At this point we have already received all of the batches, we must then merge
	// all these batches and sort them to have a deterministic ordering of transactions.
	// - Within in given batch, transactions are executed linearly, so partial sort is already done.
	// - Batches are sorted based on their first transaction's id (hash), sorted alphanumerically
	case strings.HasPrefix(line, "BATCHES_END"):
		err = ctx.readBatchesEnd()

	// this occurs when a given block is full (frozen),
	case strings.HasPrefix(line, "BLOCK_END"):
		err = ctx.readBlockEnd(line)

	// this occurs when there is a failure in executing a given block
	case strings.HasPrefix(line, "BLOCK_FAILED"):
		err = ctx.readBlockFailed(line)

	// this occurs when the root of the active banks has been computed
	case strings.HasPrefix(line, "BLOCK_ROOT"):
		err = ctx.readBlockRoot(line)

	default:
		zlog.Warn("unknown log line", zap.String("line", line))
	}
	return
}

func (r *ConsoleReader) formatError(line string, err error) error {
	chunks := strings.SplitN(line, " ", 2)
	return fmt.Errorf("%s: %s (line %q)", chunks[0], err, line)
}

func (ctx *parseCtx) readBatchFile(line string) (err error) {
	chunks := strings.Split(line, " ")
	if len(chunks) != 2 {
		return fmt.Errorf("read batch file: expected 2 fields, got %d", len(chunks))
	}

	filename := chunks[1]
	filePath := filepath.Join(ctx.batchFilesPath, filename)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf(": %w", err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			zlog.Warn("read batch file: failed to close file", zap.String("file_path", filePath))
		}
		if err := os.Remove(filePath); err != nil {
			zlog.Warn("read batch file: failed to delete file", zap.String("file_path", filePath))
		}
	}()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read batch: read all: %w", err)
	}

	batch := &pbcodec.Batch{}
	err = proto.Unmarshal(data, batch)
	if err != nil {
		return fmt.Errorf("read batch: proto unmarshall: %w", err)
	}

	for _, tx := range batch.Transactions {
		for _, i := range tx.Instructions {
			if i.ProgramId == "Vote111111111111111111111111111111111111111" {
				i.AccountChanges = nil
			}
		}
	}

	ctx.activeBank.batchAggregator = append(ctx.activeBank.batchAggregator, batch.Transactions)

	// TODO: do the fixups, `depth` setting, addition of the `Slot` and other data
	// that is not written by the batch writer.

	return nil
}

const (
	BlockWorkChunkSize   = 14
	BlockEndChunkSize    = 5
	BlockFailedChunkSize = 3
	BlockRootChunkSize   = 2
	InitChunkSize        = 3
	SlotBoundChunkSize   = 3
)

type bank struct {
	parentSlotNum   uint64
	processTrxCount uint64
	previousSlotID  string
	transactionIDs  []string
	blk             *pbcodec.Block
	ended           bool
	batchAggregator [][]*pbcodec.Transaction
}

func newBank(blockNum, parentSlotNumber uint64, previousSlotID string) *bank {
	return &bank{
		parentSlotNum:   parentSlotNumber,
		previousSlotID:  previousSlotID,
		transactionIDs:  []string{},
		batchAggregator: [][]*pbcodec.Transaction{},
		blk: &pbcodec.Block{
			Version:           1,
			Number:            blockNum,
			PreviousId:        previousSlotID,
			PreviousBlockSlot: parentSlotNumber,
		},
	}
}

// the goal is to sort the batches based on the first transaction id of each batch
func (b *bank) processBatchAggregation() error {
	indexMap := map[string]int{}
	for idx, trxID := range b.transactionIDs {
		indexMap[trxID] = idx
	}

	b.blk.Transactions = make([]*pbcodec.Transaction, len(b.transactionIDs))
	b.blk.TransactionCount = uint32(len(b.transactionIDs))

	var count int
	for _, transactions := range b.batchAggregator {
		for _, trx := range transactions {
			trxIndex := indexMap[trx.Id]
			trx.Index = uint64(trxIndex)
			count++
			b.blk.Transactions[trxIndex] = trx
		}
	}

	b.batchAggregator = [][]*pbcodec.Transaction{}

	if count != len(b.transactionIDs) {
		return fmt.Errorf("transaction ids received on BLOCK_WORK did not match the number of transactions collection from batch executions, counted %d execution, expected %d from ids", count, len(b.transactionIDs))
	}

	return nil
}

func (b *bank) getActiveTransaction(batchNum uint64, trxID string) (*pbcodec.Transaction, error) {
	length := len(b.batchAggregator[batchNum])
	if length == 0 {
		return nil, fmt.Errorf("unable to retrieve transaction trace on an empty batch")
	}
	trx := b.batchAggregator[batchNum][length-1]
	if trx.Id != trxID {
		return nil, fmt.Errorf("transaction trace ID doesn't match expected value: %s", trxID)
	}

	return trx, nil
}

// BATCHES_END
func (ctx *parseCtx) readBatchesEnd() (err error) {
	if ctx.activeBank == nil {
		return fmt.Errorf("received batch end while no active bank in context")
	}

	return nil
}

func (ctx *parseCtx) readInit(line string) (err error) {
	zlog.Debug("reading init", zap.String("line", line))
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != InitChunkSize {
		return fmt.Errorf("expected %d fields got %d", InitChunkSize, len(chunks))
	}

	var version uint64
	if version, err = strconv.ParseUint(chunks[2], 10, 64); err != nil {
		return fmt.Errorf("version to int: %w", err)
	}

	if version != 2 {
		return fmt.Errorf("unsupported DMLOG version %d, expected version 2", version)
	}

	return nil
}

// BLOCK_WORK 55295937 55295938 full Dpt1ohisw1neR8KetzS14LtY9yjq37Q3bAoowGJ5tfSA 224 161 200 0 0 0 Dpt1ohisw1neR8KetzS14LtY9yjq37Q3bAoowGJ5tfSA 0 T;trxid1;trxid2
// BLOCK_WORK PREVIOUS_BLOCK_NUM BLOCK_NUM <full/partial> PARENT_SLOT_ID NUM_ENTRIES NUM_TXS NUM_SHRED PROGRESS_NUM_ENTRIES PROGRESS_NUM_TXS PROGRESS_NUM_SHREDS PROGESS_LAST_ENTRY PROGRESS_TICK_HASH_COUNT T;TRANSACTION_IDS_VECTOR_SPLIT_BY_;
func (ctx *parseCtx) readBlockWork(line string) (err error) {
	zlog.Debug("reading block work", zap.String("line", line))
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != BlockWorkChunkSize {
		return fmt.Errorf("expected %d fields got %d", BlockWorkChunkSize, len(chunks))
	}

	var blockNum, parentSlotNumber int
	if blockNum, err = strconv.Atoi(chunks[2]); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if parentSlotNumber, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("parent slot num to int: %w", err)
	}

	previousSlotID := chunks[4]

	var b *bank
	var found bool
	if b, found = ctx.banks[uint64(blockNum)]; !found {
		zlog.Info("creating a new bank",
			zap.Int("parent_slot_number", parentSlotNumber),
			zap.Int("slot_number", blockNum),
		)
		b = newBank(uint64(blockNum), uint64(parentSlotNumber), previousSlotID)
		ctx.banks[uint64(blockNum)] = b
	}

	for _, trxID := range strings.Split(chunks[13], ";") {
		if trxID == "" || trxID == "T" {
			continue
		}
		b.transactionIDs = append(b.transactionIDs, trxID)
	}

	ctx.activeBank = b
	return nil
}

// BLOCK_END 55295941 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz 1606487316 1606487316
// BLOCK_END BLOCK_NUM LAST_ENTRY_HASH GENESIS_UNIX_TIMESTAMP CLOCK_UNIX_TIMESTAMP
func (ctx *parseCtx) readBlockEnd(line string) (err error) {
	zlog.Debug("reading block end", zap.String("line", line))

	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != BlockEndChunkSize {
		return fmt.Errorf("expected %d fields, got %d", BlockEndChunkSize, len(chunks))
	}

	var blockNum, clockTimestamp, genesisTimestamp uint64
	if blockNum, err = strconv.ParseUint(chunks[1], 10, 64); err != nil {
		return fmt.Errorf("slotNumber to int: %w", err)
	}

	if clockTimestamp, err = strconv.ParseUint(chunks[3], 10, 64); err != nil {
		return fmt.Errorf("error decoding sysvar::clock timestamp in seconds: %w", err)
	}

	if genesisTimestamp, err = strconv.ParseUint(chunks[4], 10, 64); err != nil {
		return fmt.Errorf("error decoding genesis timestamp in seconds: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("received slot end while no active bank in context")
	}

	if ctx.activeBank.blk.Number != blockNum {
		return fmt.Errorf("slot end's active bank does not match context's active bank")
	}

	slotID := chunks[2]
	ctx.activeBank.blk.Id = slotID
	ctx.activeBank.blk.GenesisUnixTimestamp = genesisTimestamp
	ctx.activeBank.blk.ClockUnixTimestamp = clockTimestamp
	ctx.activeBank.ended = true

	if err := ctx.activeBank.processBatchAggregation(); err != nil {
		return fmt.Errorf("sorting: %w", err)
	}

	// TODO: it'd be cleaner if this was `nil`, we need to update the tests.
	ctx.activeBank = nil

	return nil
}

// BLOCK_ROOT 6482838121
// Simply the root block number, when this block is done processing, and all of its votes are taken into account.
func (ctx *parseCtx) readBlockRoot(line string) (err error) {
	zlog.Debug("reading block root", zap.String("line", line))
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != BlockRootChunkSize {
		return fmt.Errorf("expected %d fields got %d", BlockRootChunkSize, len(chunks))
	}

	var rootBlock uint64
	if rootBlock, err = strconv.ParseUint(chunks[1], 10, 64); err != nil {
		return fmt.Errorf("root block num num to int: %w", err)
	}

	for bankSlotNum, bank := range ctx.banks {
		if !bank.ended {
			if bankSlotNum < rootBlock {
				zlog.Info("purging un-ended banks", zap.Uint64("purge_bank_slot", bankSlotNum), zap.Uint64("root_block", rootBlock))
				delete(ctx.banks, bankSlotNum)
			}
			continue
		}

		if rootBlock == bank.blk.Number {
			return fmt.Errorf("invalid root for bank. Root block %d cannot equal bank block number %d", rootBlock, bank.blk.Number)
		}

		bank.blk.RootNum = rootBlock
		ctx.blockBuffer <- bank.blk

		delete(ctx.banks, bankSlotNum)
	}
	zlog.Debug("ctx bank state", zap.Int("bank_count", len(ctx.banks)))
	return nil
}

// SLOT_FAILED SLOT_NUM REASON
func (ctx *parseCtx) readBlockFailed(line string) (err error) {
	zlog.Debug("reading block failed", zap.String("line", line))
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != BlockFailedChunkSize {
		return fmt.Errorf("expected %d fields got %d", BlockFailedChunkSize, len(chunks))
	}

	var blockNum uint64
	if blockNum, err = strconv.ParseUint(chunks[1], 10, 64); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("slot failed start while no active bank in context")
	}

	if ctx.activeBank.blk.Number != blockNum {
		return fmt.Errorf("slot failed's active bank does not match context's active bank")
	}

	return fmt.Errorf("slot %d failed: %s", blockNum, chunks[2])
}
