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
	"sort"
	"strconv"
	"strings"
	"sync"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/golang/protobuf/proto"
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
	src            io.Reader
	scanner        *bufio.Scanner
	close          func()
	readBuffer     chan string
	done           chan interface{}
	ctx            *parseCtx
	batchFilesPath string
}

func NewConsoleReader(reader io.Reader, batchFilesPath string, opts ...ConsoleReaderOption) (*ConsoleReader, error) {
	l := &ConsoleReader{
		src:   reader,
		close: func() {},
		ctx:   newParseCtx(batchFilesPath),
		done:  make(chan interface{}),
	}

	for _, opt := range opts {
		opt.apply(l)
	}

	l.setupScanner()
	return l, nil
}

func newScanner(reader io.Reader) *bufio.Scanner {
	buf := make([]byte, MaxTokenSize)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(buf, len(buf))
	return scanner
}

func scan(scanner *bufio.Scanner, handleLine func(line string) error) error {
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "DMLOG ") {
			continue
		}
		err := handleLine(line)
		if err != nil {
			return fmt.Errorf("scan: handle line: %w", err)
		}
	}
	return nil
}

func (l *ConsoleReader) setupScanner() {
	l.scanner = newScanner(l.src)
	l.readBuffer = make(chan string, 2000)

	go func() {
		err := scan(l.scanner, func(line string) error {
			l.readBuffer <- line
			return nil
		})

		if err == nil {
			err = l.scanner.Err()
		}

		if err != nil && err != io.EOF {
			zlog.Error("console read line scanner encountered an error", zap.Error(err))
		}

		close(l.readBuffer)
		close(l.done)
	}()
}

func (l *ConsoleReader) Done() <-chan interface{} {
	return l.done
}

func (l *ConsoleReader) Close() {
	l.close()
}

type parseCtx struct {
	activeBank        *bank
	banks             map[uint64]*bank
	conversionOptions []conversionOption
	slotBuffer        chan *pbcodec.Slot
	batchWG           sync.WaitGroup
	batchFilesPath    string
}

func newParseCtx(batchFilesPath string) *parseCtx {
	return &parseCtx{
		banks:          map[uint64]*bank{},
		slotBuffer:     make(chan *pbcodec.Slot, 10000),
		batchFilesPath: batchFilesPath,
	}
}

func (l *ConsoleReader) Read() (out interface{}, err error) {
	ctx := l.ctx
	for {
		select {
		case s := <-ctx.slotBuffer:
			return s, nil
		default:
		}

		line, ok := <-l.readBuffer
		if !ok {
			if l.scanner.Err() == nil {
				return nil, io.EOF
			}
			return nil, l.scanner.Err()
		}

		line = line[6:]

		if err = parseLine(ctx, line); err != nil {
			return nil, l.formatError(line, err)
		}
	}
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

	// when processing a block you will have SLOT_BOUNDS (between SLOT_WORK & SLOT_END) for each SLOT in that BLOCK.
	case strings.HasPrefix(line, "SLOT_BOUND"):
		err = ctx.readSlotBound(line)

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

func (l *ConsoleReader) formatError(line string, err error) error {
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
		_ = file.Close()
		if err := os.Remove(filePath); err != nil {
			zlog.Warn("failed to delete file", zap.String("file_path", filePath))
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
	blockNum        uint64
	parentSlotNum   uint64
	trxCount        uint64
	processTrxCount uint64
	previousSlotID  string
	slots           []*pbcodec.Slot
	blk             *pbcodec.Block
	sortedTrx       []*pbcodec.Transaction
	ended           bool
	batchAggregator [][]*pbcodec.Transaction
}

func newBank(blockNum, parentSlotNumber, blockHeight uint64, previousSlotID string) *bank {
	return &bank{
		blockNum:        blockNum,
		parentSlotNum:   parentSlotNumber,
		previousSlotID:  previousSlotID,
		slots:           []*pbcodec.Slot{},
		sortedTrx:       []*pbcodec.Transaction{},
		batchAggregator: [][]*pbcodec.Transaction{},
		blk: &pbcodec.Block{
			Number:            blockNum,
			Height:            blockHeight,
			PreviousId:        previousSlotID,
			PreviousBlockSlot: parentSlotNumber,
		},
	}
}

// the goal is to sort the batches based on the first transaction id of each batch
func (b *bank) sortTrx() {
	type batchSort struct {
		index int
		trxID string
	}
	batches := make([]*batchSort, len(b.batchAggregator))
	// the batch num starts at 0 and is increment, thus
	// it can be used as our array index
	for i, transactions := range b.batchAggregator {
		batches[i] = &batchSort{
			index: i,
			trxID: transactions[0].Id,
		}
	}
	sort.Slice(batches, func(i, j int) bool {
		return strings.Compare(batches[i].trxID, batches[j].trxID) < 0
	})

	for _, batch := range batches {
		b.sortedTrx = append(b.sortedTrx, b.batchAggregator[batch.index]...)
	}

	b.batchAggregator = [][]*pbcodec.Transaction{}
}

func (b *bank) registerSlot(slotNum uint64, slotID string) {
	s := b.createSlot(slotNum, slotID)
	b.slots = append(b.slots, s)
}

func (b *bank) createSlot(slotNum uint64, slotID string) *pbcodec.Slot {
	s := &pbcodec.Slot{
		Id:               slotID,
		Number:           slotNum,
		PreviousId:       b.previousSlotID,
		Version:          1,
		TransactionCount: uint32(len(b.sortedTrx)),
	}

	for idx, trx := range b.sortedTrx {
		trx.Index = uint64(idx)
		trx.SlotNum = slotNum
		trx.SlotHash = slotID
		s.Transactions = append(s.Transactions, trx)
	}
	b.previousSlotID = slotID
	b.sortedTrx = []*pbcodec.Transaction{}
	return s
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

	ctx.activeBank.sortTrx()
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

// BLOCK_WORK 55295937 55295938 full Dpt1ohisw1neR8KetzS14LtY9yjq37Q3bAoowGJ5tfSA 51936823 224 161 200 0 0 0 Dpt1ohisw1neR8KetzS14LtY9yjq37Q3bAoowGJ5tfSA 0
// BLOCK_WORK PREVIOUS_BLOCK_NUM BLOCK_NUM <full/partial> PARENT_SLOT_ID BLOCK_HEIGHT NUM_ENTRIES NUM_TXS NUM_SHRED PROGRESS_NUM_ENTRIES PROGRESS_NUM_TXS PROGRESS_NUM_SHREDS PROGESS_LAST_ENTRY PROGRESS_TICK_HASH_COUNT
func (ctx *parseCtx) readBlockWork(line string) (err error) {
	zlog.Debug("reading block work", zap.String("line", line))
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != BlockWorkChunkSize {
		return fmt.Errorf("expected %d fields got %d", BlockWorkChunkSize, len(chunks))
	}

	var blockNum, parentSlotNumber, blockHeight, trxCount int
	if blockNum, err = strconv.Atoi(chunks[2]); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if parentSlotNumber, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("parent slot num to int: %w", err)
	}

	if blockHeight, err = strconv.Atoi(chunks[5]); err != nil {
		return fmt.Errorf("parent slot num to int: %w", err)
	}

	if trxCount, err = strconv.Atoi(chunks[6]); err != nil {
		return fmt.Errorf("transaction count: %w", err)
	}

	previousSlotID := chunks[4]

	var b *bank
	var found bool
	if b, found = ctx.banks[uint64(blockNum)]; !found {
		zlog.Info("creating a new bank",
			zap.Int("parent_slot_number", parentSlotNumber),
			zap.Int("slot_number", blockNum),
		)
		b = newBank(uint64(blockNum), uint64(parentSlotNumber), uint64(blockHeight), previousSlotID)
		ctx.banks[uint64(blockNum)] = b
	}
	b.trxCount = b.trxCount + uint64(trxCount)

	ctx.activeBank = b
	return nil
}

// SLOT_BOUND 55295941 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz
// SLOT_BOUND BLOCK_NUM SLOT_ID
func (ctx *parseCtx) readSlotBound(line string) (err error) {
	zlog.Debug("reading slot bound", zap.String("line", line))
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != SlotBoundChunkSize {
		return fmt.Errorf("expected %d fields got %d", SlotBoundChunkSize, len(chunks))
	}

	var blockNum uint64
	if blockNum, err = strconv.ParseUint(chunks[1], 10, 64); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("received slot bound while no active bank in context")
	}

	slotId := chunks[2]
	ctx.activeBank.registerSlot(blockNum, slotId)
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

	if ctx.activeBank.blockNum != blockNum {
		return fmt.Errorf("slot end's active bank does not match context's active bank")
	}

	slotID := chunks[2]
	ctx.activeBank.blk.Id = slotID
	ctx.activeBank.blk.GenesisUnixTimestamp = genesisTimestamp
	ctx.activeBank.blk.ClockUnixTimestamp = clockTimestamp
	ctx.activeBank.ended = true
	// TODO: it'd be cleaner if this was `nil`, we need to update the tests.
	ctx.activeBank = nil

	return nil
}

// BLOCK_ROOT 6482838121
// Simply the root block number, when this block is done processing, and all of its votes are taken into account.
func (ctx *parseCtx) readBlockRoot(line string) (err error) {
	zlog.Debug("reading block root failed", zap.String("line", line))
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

		bank.blk.RootNum = rootBlock
		for _, slot := range bank.slots {
			slot.Block = bank.blk
			for i, t := range slot.Transactions {
				t.Index = uint64(i)
				t.SlotHash = slot.Id
				t.SlotNum = slot.Number
			}

			if len(ctx.slotBuffer) == cap(ctx.slotBuffer) {
				return fmt.Errorf("unable to put slot in buffer reached buffer capacity size %q", cap(ctx.slotBuffer))
			}
			ctx.slotBuffer <- slot
		}
		delete(ctx.banks, bankSlotNum)
	}
	zlog.Info("ctx bank state", zap.Int("bank_count", len(ctx.banks)))
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

	if ctx.activeBank.blockNum != blockNum {
		return fmt.Errorf("slot failed's active bank does not match context's active bank")
	}

	return fmt.Errorf("slot %d failed: %s", blockNum, chunks[2])
}
