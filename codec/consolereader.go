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
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
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

	ctx          *parseCtx
	maxTokenSize uint64
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

func newScanner(maxTokenSize uint64, reader io.Reader) *bufio.Scanner {
	buf := make([]byte, maxTokenSize)
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
	l.scanner = newScanner(l.maxTokenSize, l.src)
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

	case strings.HasPrefix(line, "BATCH_END"):
		err = ctx.readBatchEnd(line)

	case strings.HasPrefix(line, "SLOT_END"):
		err = ctx.readSlotEnd(line)

	case strings.HasPrefix(line, "SLOT_FAILED"):
		err = ctx.readSlotFailed(line)

	case strings.HasPrefix(line, "TRX_START"):
		err = ctx.readTransactionStart(line)

	case strings.HasPrefix(line, "TRX_END"):
		err = ctx.readTransactionEnd(line)

	case strings.HasPrefix(line, "TRX_LOG"):
		err = ctx.readTransactionLog(line)

	case strings.HasPrefix(line, "INST_S"):
		err = ctx.readInstructionStart(line)

	case strings.HasPrefix(line, "ACCT_CH"):
		err = ctx.readAccountChange(line)

	case strings.HasPrefix(line, "LAMP_CH"):
		err = ctx.readLamportsChange(line)

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

	dmlogScanner := newScanner(MaxTokenSize, file)
	err = scan(dmlogScanner, func(line string) error {
		return parseLine(ctx, line)
	})

	if err == nil {
		err = dmlogScanner.Err()
	}

	if err != nil {
		log.Fatal(err)
	}

	return nil
}

const (
	SlotWorkChunkSize   = 14
	SlotEndChunkSize    = 5
	SlotBoundChunkSize  = 3
	SlotFailedChunkSize = 3
	TrxStartChunkSize   = 8
	TrxLogChunkSize     = 4
	InsStartChunkSize   = 8
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

	batchAggregator map[uint64][]*pbcodec.Transaction
}

func newBank(blockNum, parentSlotNumber, blockHeight uint64, previousSlotID string) *bank {
	return &bank{
		blockNum:        blockNum,
		parentSlotNum:   parentSlotNumber,
		previousSlotID:  previousSlotID,
		slots:           []*pbcodec.Slot{},
		sortedTrx:       []*pbcodec.Transaction{},
		batchAggregator: map[uint64][]*pbcodec.Transaction{},
		blk: &pbcodec.Block{
			Number:            blockNum,
			Height:            blockHeight,
			PreviousId:        previousSlotID,
			PreviousBlockSlot: parentSlotNumber,
		},
	}
}

func (b *bank) sortTrx() {
	// the goal is to sort the batches based on the first transaction id of each batch

	type batchSort struct {
		batchNum uint64
		trxID    string
	}
	batches := make([]*batchSort, len(b.batchAggregator))
	// the batch num starts at 0 and is increment, thus
	// it can be used as our array index
	for batchNum, transactions := range b.batchAggregator {
		batches[batchNum] = &batchSort{
			batchNum: batchNum,
			// can a batch have no transactions in it? I don't think so...
			trxID: transactions[0].Id,
		}
	}
	sort.Slice(batches, func(i, j int) bool {
		return strings.Compare(batches[i].trxID, batches[j].trxID) < 0
	})

	for _, batch := range batches {
		b.sortedTrx = append(b.sortedTrx, b.batchAggregator[batch.batchNum]...)
	}

	b.batchAggregator = map[uint64][]*pbcodec.Transaction{}
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
		trx.SlotNum = uint64(slotNum)
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
func (ctx *parseCtx) readBatchEnd(line string) (err error) {
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

// TRX_START 0 3JwX7ifk5BYZWdBK1o9Zs4wEZ6HP8MWbxhgZD7u1PzSDbaDLrZZbhBnvQJsVMPpWdpaTAFiUiQWZcbEdc3Nfj9Sq 1 0 3 3rqEEEGjHRyndHuduBcjkf17rX3hgmGACpYTQYeZ5Ltk:8xV77wuFP5BkMDdb1845hRRWZNbDNAbcV75BjMuViWpf:SysvarS1otHashes111111111111111111111111111:SysvarC1ock11111111111111111111111111111111:Vote111111111111111111111111111111111111111 2pE6pkNJzuMz4r8owVi4hrCctEvGyrg1g3SLD4nbcsxz
// TRX_START BATCH_NUM SIG1:SIG2:SIG3 NUM_REQUIRED_SIGN NUM_READONLY_SIGN_ACT NUM_READONLY_UNSIGNED_ACT ACTKEY1:ACTKEY2:ACTKEY3 RECENT_BLOCKHASH
func (ctx *parseCtx) readTransactionStart(line string) (err error) {
	chunks := strings.Split(line, " ")
	if len(chunks) != TrxStartChunkSize {
		return fmt.Errorf("expected %d fields got %d", TrxStartChunkSize, len(chunks))
	}

	var batchNum, roSignedAccts, roUnsignedAccts, reqSigs int

	if batchNum, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("received transaction start while no active bank in context")
	}

	sigs := strings.Split(chunks[2], ":")
	id := sigs[0]
	additionalSigs := sigs[1:]

	if reqSigs, err = strconv.Atoi(chunks[3]); err != nil {
		return fmt.Errorf("failed decoding num_required_signatures: %w", err)
	}
	if roSignedAccts, err = strconv.Atoi(chunks[4]); err != nil {
		return fmt.Errorf("failed decoding num_readonly_signed_accounts: %w", err)
	}
	if roUnsignedAccts, err = strconv.Atoi(chunks[5]); err != nil {
		return fmt.Errorf("failed decoding num_readonly_unsigned_accounts: %w", err)
	}

	accountKeys := strings.Split(chunks[6], ":")
	recentBlockHash := chunks[7]
	transaction := &pbcodec.Transaction{
		Id:                   id,
		AdditionalSignatures: additionalSigs,
		AccountKeys:          accountKeys,
		Header: &pbcodec.MessageHeader{
			NumRequiredSignatures:       uint32(reqSigs),
			NumReadonlySignedAccounts:   uint32(roSignedAccts),
			NumReadonlyUnsignedAccounts: uint32(roUnsignedAccts),
		},
		RecentBlockhash: recentBlockHash,
	}

	ctx.activeBank.recordTransaction(uint64(batchNum), transaction)

	return nil
}

// TRX_END 55295915 51noDfuFBWCvunwvNrUKL4gHVZLDrAp4ihdecoNRirD5zdLsxSn3PZWj1bLQ52rQ9MpAKgbNyd76sdnzq5CM6cFG
// TRX_END SLOT_NUM TX_SIGNATURE
func (ctx *parseCtx) readTransactionEnd(line string) error {
	return nil
}

// TRX_L BATCH_NUM TX_SIGNATURE LOG_IN_HEX
func (ctx *parseCtx) readTransactionLog(line string) (err error) {
	chunks := strings.Split(line, " ")
	if len(chunks) != TrxLogChunkSize {
		return fmt.Errorf("expected %d fields got %d", TrxLogChunkSize, len(chunks))
	}

	var batchNum int
	if batchNum, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("received transaction log while no active bank in context")
	}

	id := chunks[2]
	logLine, err := hex.DecodeString(chunks[3])
	if err != nil {
		return fmt.Errorf("log line failed hex decoding: %w", err)
	}

	if err = ctx.activeBank.recordLogMessage(uint64(batchNum), id, string(logLine)); err != nil {
		return fmt.Errorf("read trx log message: %w", err)
	}
	return nil
}

// INST_S 0 S5eYZCYnXoa3858MJ2cvdXCXRW8xiTagWXM4WNggt96A5qm2NoHtYro56GGwygCgfKJzN733PxMBEEH7TAoHRYh 1 0 Vote111111111111111111111111111111111111111 020000000200000000000000adbf4b0300000000aebf4b03000000000e060473b5c277d1949ddc92c7a92e2d835008d70fd4817b5110611c11d52aa801c214d85f00000000 Vote111111111111111111111111111111111111111:00;9SE5oHdQ88rVFPcJZjn7fNGSXhU7JQfZ5Vks1h5VNCWj:01;SysvarS1otHashes111111111111111111111111111:00;SysvarC1ock11111111111111111111111111111111:00;AHg5MDTTPKvfCxYy8Zb3NpRYG7ixsx2uTT1MUs7DwzEu:11
// INST_S BATCH_NUM TRX_ID ORDINAL PARENT_ORDINAL PROGRAM_ID DATA ACCOUNTS
func (ctx *parseCtx) readInstructionStart(line string) (err error) {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != InsStartChunkSize {
		return fmt.Errorf("expected %d fields got %d", InsStartChunkSize, len(chunks))
	}

	if len(chunks) != 8 {
		return fmt.Errorf("read instructionTrace start: expected 8 fields, got %d", len(chunks))
	}

	var batchNum int
	if batchNum, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("received instruction start log while no active bank in context")
	}

	id := chunks[2]
	ordinal, err := strconv.Atoi(chunks[3])
	if err != nil {
		return fmt.Errorf("read instructionTrace start: ordinal to int: %w", err)
	}

	parentOrdinal, err := strconv.Atoi(chunks[4])
	if err != nil {
		return fmt.Errorf("read instructionTrace start: parent ordinal to int: %w", err)
	}

	program := chunks[5]
	data := chunks[6]
	hexData, err := hex.DecodeString(data)
	if err != nil {
		return fmt.Errorf("read instructionTrace start: hex decode data: %w", err)
	}

	var accountKeys []string
	accounts := strings.Split(chunks[7], ";")
	for _, acct := range accounts {
		accountKeys = append(accountKeys, strings.Split(acct, ":")[0])
	}

	instruction := &pbcodec.Instruction{
		ProgramId:     program,
		Data:          hexData,
		Ordinal:       uint32(ordinal),
		ParentOrdinal: uint32(parentOrdinal),
		AccountKeys:   accountKeys,
	}

	err = ctx.activeBank.recordInstruction(uint64(batchNum), id, instruction)
	if err != nil {
		return fmt.Errorf("read instructionTrace start: %w", err)
	}

	return nil
}

// ACCT_CH 0 4YU3GFLmzR7b58YDgNCwHD3YfEHTLq7b13gSr3zWHWa4W7FuvBrWQgLnvQT4kfxJ5ZTULokJK7x2d7nfKU3UWd8i 1 2xjAQsHLsV36NLFkxdApzLg4SNqm15mNqYaBQ4xp5joh 01000000ed76cf23520b41e64596066fc8dbf63e94e1b5e97add78d9501f796142b17e95ed76cf23520b41e64596066fc8dbf63e94e1b5e97add78d9501f796142b17e950a1f000000000000007abf4b03000000001f0000007bbf4b03000000001e0000007cbf4b03000000001d0000008cbf4b03000000001c0000008dbf4b03000000001b0000008ebf4b03000000001a0000008fbf4b03000000001900000091bf4b03000000001800000092bf4b03000000001700000093bf4b03000000001600000094bf4b03000000001500000095bf4b03000000001400000096bf4b03000000001300000097bf4b03000000001200000098bf4b03000000001100000099bf4b0300000000100000009abf4b03000000000f0000009bbf4b03000000000e0000009cbf4b03000000000d0000009dbf4b03000000000c0000009ebf4b03000000000b0000009fbf4b03000000000a000000a0bf4b030000000009000000a1bf4b030000000008000000a2bf4b030000000007000000a3bf4b030000000006000000a4bf4b030000000005000000a5bf4b030000000004000000a6bf4b030000000003000000a7bf4b030000000002000000a8bf4b0300000000010000000179bf4b030000000001000000000000007f00000000000000ed76cf23520b41e64596066fc8dbf63e94e1b5e97add78d9501f796142b17e950000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001f000000000000000112000000000000006e00000000000000fa9601000000000000000000000000006f00000000000000e175070000000000fa9601000000000070000000000000002c7b0d0000000000e175070000000000710000000000000000531300000000002c7b0d0000000000720000000000000093081900000000000053130000000000730000000000000026cb1e000000000093081900000000007400000000000000b60624000000000026cb1e00000000007500000000000000b6fb280000000000b6062400000000007600000000000000a9332e0000000000b6fb28000000000077000000000000002631330000000000a9332e00000000007800000000000000ba4738000000000026313300000000007900000000000000219b3c0000000000ba473800000000007a0000000000000025a6410000000000219b3c00000000007b00000000000000f71847000000000025a64100000000007c00000000000000e68f4c0000000000f7184700000000007d0000000000000075d74f0000000000e68f4c00000000007e00000000000000e78253000000000075d74f00000000007f000000000000000c96570000000000e782530000000000a8bf4b0300000000bf14d85f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 01000000ed76cf23520b41e64596066fc8dbf63e94e1b5e97add78d9501f796142b17e95ed76cf23520b41e64596066fc8dbf63e94e1b5e97add78d9501f796142b17e950a1f000000000000007bbf4b03000000001f0000007cbf4b03000000001e0000008cbf4b03000000001d0000008dbf4b03000000001c0000008ebf4b03000000001b0000008fbf4b03000000001a00000091bf4b03000000001900000092bf4b03000000001800000093bf4b03000000001700000094bf4b03000000001600000095bf4b03000000001500000096bf4b03000000001400000097bf4b03000000001300000098bf4b03000000001200000099bf4b0300000000110000009abf4b0300000000100000009bbf4b03000000000f0000009cbf4b03000000000e0000009dbf4b03000000000d0000009ebf4b03000000000c0000009fbf4b03000000000b000000a0bf4b03000000000a000000a1bf4b030000000009000000a2bf4b030000000008000000a3bf4b030000000007000000a4bf4b030000000006000000a5bf4b030000000005000000a6bf4b030000000004000000a7bf4b030000000003000000a8bf4b030000000002000000a9bf4b030000000001000000017abf4b030000000001000000000000007f00000000000000ed76cf23520b41e64596066fc8dbf63e94e1b5e97add78d9501f796142b17e950000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001f000000000000000112000000000000006e00000000000000fa9601000000000000000000000000006f00000000000000e175070000000000fa9601000000000070000000000000002c7b0d0000000000e175070000000000710000000000000000531300000000002c7b0d0000000000720000000000000093081900000000000053130000000000730000000000000026cb1e000000000093081900000000007400000000000000b60624000000000026cb1e00000000007500000000000000b6fb280000000000b6062400000000007600000000000000a9332e0000000000b6fb28000000000077000000000000002631330000000000a9332e00000000007800000000000000ba4738000000000026313300000000007900000000000000219b3c0000000000ba473800000000007a0000000000000025a6410000000000219b3c00000000007b00000000000000f71847000000000025a64100000000007c00000000000000e68f4c0000000000f7184700000000007d0000000000000075d74f0000000000e68f4c00000000007e00000000000000e78253000000000075d74f00000000007f000000000000000d96570000000000e782530000000000a9bf4b0300000000bf14d85f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
// ACCT_CH BATCH_NUM TRX_ID ORDINAL PUBKEY PREV_DATA NEW_DATA
func (ctx *parseCtx) readAccountChange(line string) (err error) {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 7 {
		return fmt.Errorf("read account change: expected 7 fields, got %d", len(chunks))
	}

	var batchNum, ordinal int
	if batchNum, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("received account change log while no active bank in context")
	}

	if ordinal, err = strconv.Atoi(chunks[3]); err != nil {
		return fmt.Errorf("read account change: ordinal to int: %w", err)
	}

	trxID := chunks[2]
	pubKey := chunks[4]
	var prevData, newData []byte

	if prevData, err = hex.DecodeString(chunks[5]); err != nil {
		return fmt.Errorf("read account change: hex decode prev data: %w", err)
	}

	if newData, err = hex.DecodeString(chunks[6]); err != nil {
		return fmt.Errorf("read account change: hex decode new data: %w", err)
	}

	accountChange := &pbcodec.AccountChange{
		Pubkey:        pubKey,
		PrevData:      prevData,
		NewData:       newData,
		NewDataLength: uint64(len(newData)),
	}

	if err = ctx.activeBank.recordAccountChange(uint64(batchNum), trxID, ordinal, accountChange); err != nil {
		return fmt.Errorf("read account change: %w", err)
	}

	return nil
}

// LAMP_CH 0 aaa 61hY5LpNSSH3zpnxoLYf5pmStN4JRMJ8H4nt4omyNQgaBb78APUetZRw23QdWpZLWF22KG1rBvNdX9XJcut21HQZ 1 11111111111111111111111111111111 499999892500 494999892500
// LAMP_CH BATCH_NUM TRX_ID OWNER 1 11111111111111111111111111111111 499999892500 494999892500
func (ctx *parseCtx) readLamportsChange(line string) (err error) {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 7 {
		return fmt.Errorf("read lamport change: expected 7 fields, got %d", len(chunks))
	}

	var batchNum, ordinal, prevLamports, newLamports int
	if batchNum, err = strconv.Atoi(chunks[1]); err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	if ctx.activeBank == nil {
		return fmt.Errorf("received lamport change log while no active bank in context")
	}

	if ordinal, err = strconv.Atoi(chunks[3]); err != nil {
		return fmt.Errorf("read lamport change: ordinal to int: %w", err)
	}

	trxID := chunks[2]
	owner := chunks[4]

	if prevLamports, err = strconv.Atoi(chunks[5]); err != nil {
		return fmt.Errorf("read lamport change: hex decode prev lamports data: %w", err)
	}

	if newLamports, err = strconv.Atoi(chunks[6]); err != nil {
		return fmt.Errorf("read lamport change: hex decode new lamports data: %w", err)
	}

	balanceChange := &pbcodec.BalanceChange{
		Pubkey:       owner,
		PrevLamports: uint64(prevLamports),
		NewLamports:  uint64(newLamports),
	}

	if err = ctx.activeBank.recordLamportsChange(uint64(batchNum), trxID, ordinal, balanceChange); err != nil {
		return fmt.Errorf("read lamports change: %w", err)
	}

	return nil
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
