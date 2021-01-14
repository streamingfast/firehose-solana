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
	src        io.Reader
	scanner    *bufio.Scanner
	close      func()
	readBuffer chan string
	done       chan interface{}
	ctx        *parseCtx
}

func NewConsoleReader(reader io.Reader, opts ...ConsoleReaderOption) (*ConsoleReader, error) {
	l := &ConsoleReader{
		src:   reader,
		close: func() {},
		ctx:   newParseCtx(),
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
}

func newParseCtx() *parseCtx {
	return &parseCtx{
		banks:      map[uint64]*bank{},
		slotBuffer: make(chan *pbcodec.Slot, 100),
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
	case strings.HasPrefix(line, "BATCH"):
		err = ctx.readBatchFile(line)

	case strings.HasPrefix(line, "SLOT_WORK"):
		err = ctx.readSlotWork(line)

	case strings.HasPrefix(line, "SLOT_BOUND"):
		err = ctx.readSlotBound(line)

	case strings.HasPrefix(line, "BATCHES_END"):
		err = ctx.readBatchEnd()

	case strings.HasPrefix(line, "SLOT_END"):
		err = ctx.readSlotEnd(line)

	case strings.HasPrefix(line, "SLOT_FAILED"):
		err = ctx.readSlotFailed(line)

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

	file, err := os.Open(chunks[1])
	if err != nil {
		return fmt.Errorf(": %w", err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read batch: read all: %w", err)
	}

	batch := &pbcodec.Batch{}
	err = proto.Unmarshal(data, batch)
	if err != nil {
		return fmt.Errorf("read batch: proto unmarshall: %w", err)
	}

	ctx.activeBank.batchAggregator = append(ctx.activeBank.batchAggregator, batch.Transactions)

	return nil
}

const (
	SlotWorkChunkSize   = 14
	SlotEndChunkSize    = 5
	SlotBoundChunkSize  = 3
	SlotFailedChunkSize = 3
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

func (b *bank) recordTransaction(batchNum uint64, trx *pbcodec.Transaction) {
	b.batchAggregator[batchNum] = append(b.batchAggregator[batchNum], trx)
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

func (b *bank) recordLogMessage(batchNum uint64, trxID string, log string) error {
	trx, err := b.getActiveTransaction(batchNum, trxID)
	if err != nil {
		return fmt.Errorf("record log message: unable to retrieve transaction: %w", err)
	}

	trx.LogMessages = append(trx.LogMessages, log)
	return nil
}

func (b *bank) recordInstruction(batchNum uint64, trxID string, instruction *pbcodec.Instruction) error {
	trx, err := b.getActiveTransaction(batchNum, trxID)
	if err != nil {
		return fmt.Errorf("record instruction: unable to retrieve transaction: %w", err)
	}

	trx.Instructions = append(trx.Instructions, instruction)
	return nil
}

func (b *bank) recordAccountChange(batchNum uint64, trxID string, ordinal int, accountChange *pbcodec.AccountChange) error {
	trx, err := b.getActiveTransaction(batchNum, trxID)
	if err != nil {
		return fmt.Errorf("record account change: unable to retrieve transaction: %w", err)
	}

	trx.Instructions[ordinal-1].AccountChanges = append(trx.Instructions[ordinal-1].AccountChanges, accountChange)
	return nil
}

func (b *bank) recordLamportsChange(batchNum uint64, trxID string, ordinal int, balanceChange *pbcodec.BalanceChange) error {
	trx, err := b.getActiveTransaction(batchNum, trxID)
	if err != nil {
		return fmt.Errorf("record balance change: unable to retrieve transaction: %w", err)
	}

	trx.Instructions[ordinal-1].BalanceChanges = append(trx.Instructions[ordinal-1].BalanceChanges, balanceChange)
	return nil
}

// BATCH_END
func (ctx *parseCtx) readBatchEnd() (err error) {
	if ctx.activeBank == nil {
		return fmt.Errorf("received batch end while no active bank in context")
	}

	ctx.activeBank.sortTrx()
	return nil
}

// SLOT_WORK 55295937 55295938 full Dpt1ohisw1neR8KetzS14LtY9yjq37Q3bAoowGJ5tfSA 51936823 224 161 200 0 0 0 Dpt1ohisw1neR8KetzS14LtY9yjq37Q3bAoowGJ5tfSA 0
// SLOT_WORK PREVIOUS_BLOCK_NUM BLOCK_NUM <full/partial> PARENT_SLOT_ID BLOCK_HEIGHT NUM_ENTRIES NUM_TXS NUM_SHRED PROGRESS_NUM_ENTRIES PROGRESS_NUM_TXS PROGRESS_NUM_SHREDS PROGESS_LAST_ENTRY PROGRESS_TICK_HASH_COUNT
func (ctx *parseCtx) readSlotWork(line string) (err error) {
	zlog.Debug("reading slot work", zap.String("line", line))
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != SlotWorkChunkSize {
		return fmt.Errorf("expected %d fields got %d", SlotWorkChunkSize, len(chunks))
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

	var blockNum int
	if blockNum, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("received slot bound while no active bank in context")
	}

	slotId := chunks[2]
	ctx.activeBank.registerSlot(uint64(blockNum), slotId)
	return nil
}

// SLOT_END 55295941 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz 1606487316 1606487316
// SLOT_END BLOCK_NUM LAST_ENTRY_HASH GENESIS_UNIX_TIMESTAMP CLOCK_UNIX_TIMESTAMP
func (ctx *parseCtx) readSlotEnd(line string) (err error) {
	zlog.Debug("reading slot end", zap.String("line", line))

	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != SlotEndChunkSize {
		return fmt.Errorf("expected %d fields, got %d", SlotEndChunkSize, len(chunks))
	}

	var blockNum, clockTimestamp, genesisTimestamp int
	if blockNum, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("slotNumber to int: %w", err)
	}

	if clockTimestamp, err = strconv.Atoi(chunks[3]); err != nil {
		return fmt.Errorf("error decoding sysvar::clock timestamp in seconds: %w", err)
	}

	if genesisTimestamp, err = strconv.Atoi(chunks[4]); err != nil {
		return fmt.Errorf("error decoding genesis timestamp in seconds: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("received slot end while no active bank in context")
	}

	if ctx.activeBank.blockNum != uint64(blockNum) {
		return fmt.Errorf("slot end's active bank does not match context's active bank")
	}

	slotID := chunks[2]

	blk := ctx.activeBank.blk
	blk.Id = slotID
	blk.GenesisUnixTimestamp = uint64(genesisTimestamp)
	blk.ClockUnixTimestamp = uint64(clockTimestamp)
	for _, slot := range ctx.activeBank.slots {

		slot.Block = blk
		if len(ctx.slotBuffer) == cap(ctx.slotBuffer) {
			return fmt.Errorf("unable to put slot in buffer reached buffer capacity size %q", cap(ctx.slotBuffer))
		}
		ctx.slotBuffer <- slot
	}

	ctx.activeBank = nil
	delete(ctx.banks, uint64(blockNum))
	return nil
}

// SLOT_FAILED SLOT_NUM REASON
func (ctx *parseCtx) readSlotFailed(line string) (err error) {
	zlog.Debug("reading slot failed", zap.String("line", line))
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != SlotFailedChunkSize {
		return fmt.Errorf("expected %d fields got %d", SlotFailedChunkSize, len(chunks))
	}

	var blockNum int
	if blockNum, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("slot failed start while no active bank in context")
	}

	if ctx.activeBank.blockNum != uint64(blockNum) {
		return fmt.Errorf("slot failed's active bank does not match context's active bank")
	}

	return fmt.Errorf("slot %d failed: %s", blockNum, chunks[2])
}

func splitNToM(line string, min, max int) ([]string, error) {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) < min || len(chunks) > max {
		return nil, fmt.Errorf("expected between %d to %d fields (inclusively), got %d", min, max, len(chunks))
	}

	return chunks, nil
}

func (ctx *parseCtx) readDeepmindVersion(line string) error {
	chunks, err := splitNToM(line, 2, 3)
	if err != nil {
		return err
	}

	majorVersion := chunks[1]
	if !inSupportedVersion(majorVersion) {
		return fmt.Errorf("deep mind reported version %s, but this reader supports only %s", majorVersion, strings.Join(supportedVersions, ", "))
	}

	zlog.Info("read deep mind version", zap.String("major_version", majorVersion))

	return nil
}

func inSupportedVersion(majorVersion string) bool {
	for _, supportedVersion := range supportedVersions {
		if majorVersion == supportedVersion {
			return true
		}
	}

	return false
}
