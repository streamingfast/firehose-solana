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
	"os"
	"strconv"
	"strings"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"go.uber.org/zap"
)

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

	ctx *parseCtx
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

func (l *ConsoleReader) setupScanner() {
	maxTokenSize := uint64(50 * 1024 * 1024)
	if maxBufferSize := os.Getenv("MINDREADER_MAX_TOKEN_SIZE"); maxBufferSize != "" {
		bs, err := strconv.ParseUint(maxBufferSize, 10, 64)
		if err != nil {
			zlog.Error("environment variable 'MINDREADER_MAX_TOKEN_SIZE' is set but invalid parse uint", zap.Error(err))
		} else {
			zlog.Info("setting max_token_size from environment variable MINDREADER_MAX_TOKEN_SIZE", zap.Uint64("max_token_size", bs))
			maxTokenSize = bs
		}
	}
	buf := make([]byte, maxTokenSize)
	scanner := bufio.NewScanner(l.src)
	scanner.Buffer(buf, len(buf))
	l.scanner = scanner
	l.readBuffer = make(chan string, 2000)

	go func() {
		for l.scanner.Scan() {
			line := l.scanner.Text()
			if !strings.HasPrefix(line, "DMLOG ") {
				continue
			}
			l.readBuffer <- line
		}

		err := l.scanner.Err()
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
	slot     *pbcodec.Slot
	trxIndex uint64
	trxMap   map[string]*trxCtx

	conversionOptions []conversionOption
	finalized         bool
}

func newParseCtx() *parseCtx {
	return &parseCtx{
		slot:   &pbcodec.Slot{},
		trxMap: map[string]*pbcodec.Transaction{},
	}
}

func (l *ConsoleReader) Read() (out interface{}, err error) {
	ctx := l.ctx
	zlog.Debug("start reading new slot.")
	for line := range l.readBuffer {
		line = line[6:]

		if traceEnabled {
			zlog.Debug("extracing deep mind data from line", zap.String("line", line))
		}

		// Order of conditions is based (approximately) on those that will appear more often
		switch {
		case strings.HasPrefix(line, "SLOT_PROCESS"):
			err = ctx.readSlotProcess(line)

		case strings.HasPrefix(line, "SLOT_END"):
			return ctx.readSlotEnd(line)

		case strings.HasPrefix(line, "SLOT_FAILED"):
			err = ctx.readSlotFailed(line)

		case strings.HasPrefix(line, "TRX_S"):
			err = ctx.readTransactionStart(line)

		case strings.HasPrefix(line, "TRX_E"):
			err = ctx.readTransactionEnd(line)

		case strings.HasPrefix(line, "TRX_L"):
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

		if err != nil {
			return nil, l.formatError(line, err)
		}
	}

	if l.scanner.Err() == nil {
		return nil, io.EOF
	}

	return nil, l.scanner.Err()
}

func (l *ConsoleReader) formatError(line string, err error) error {
	chunks := strings.SplitN(line, " ", 2)
	return fmt.Errorf("%s: %s (line %q)", chunks[0], err, line)
}

func (ctx *parseCtx) resetSlot() {
	if ctx.slot != nil {
		ctx.resetTrx()
	}
	ctx.finalized = false
	ctx.slot = nil
}

func (ctx *parseCtx) resetTrx() {
	ctx.trxMap = map[string]*pbcodec.Transaction{}
}

func (ctx *parseCtx) readSlotProcess(line string) error {
	zlog.Debug("reading slot process:", zap.String("line", line))
	if ctx.finalized {
		ctx.resetSlot()
	}

	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 16 {
		return fmt.Errorf("read transaction provcess: expected 16 fields, got %d", len(chunks))
	}

	full := chunks[1] == "full"
	slotID := chunks[3]
	slotPreviousID := chunks[4]

	slotNumber, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("read transaction provcess: slotNumber to int: %w", err)
	}

	rootSlotNum, err := strconv.Atoi(chunks[8])
	if err != nil {
		return fmt.Errorf("read transaction provcess: slotNumber to int: %w", err)
	}

	slot := &pbcodec.Slot{
		Version:    1,
		Number:     uint64(slotNumber),
		PreviousId: slotPreviousID, //from fist full or partial
		Block:      nil,

		RootSlotNum: uint64(rootSlotNum),
	}

	if full {
		ctx.recordSlotProcessFull(slotID, slot)
	} else {
		ctx.recordSlotProcessPartial(slot)
	}

	return nil
}

func (ctx *parseCtx) recordSlotProcessFull(slotID string, slot *pbcodec.Slot) {
	if ctx.slot == nil {
		ctx.slot = slot
	}
	ctx.slot.Id = slotID
}

func (ctx *parseCtx) recordSlotProcessPartial(slot *pbcodec.Slot) {
	ctx.resetTrx()
	ctx.slot = slot
}

// SLOT_END 3 120938102938 1029830129830192
func (ctx *parseCtx) readSlotEnd(line string) (*pbcodec.Slot, error) {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 2 {
		return nil, fmt.Errorf("read slot end: expected 2 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return nil, fmt.Errorf("read slot end: slotNumber to int: %w", err)
	}

	genesisTimestamp, err := strconv.Atoi(chunks[2])
	if err != nil {
		return nil, fmt.Errorf("error decoding genesis timestamp in seconds: %w", err)
	}

	clockTimestamp, err := strconv.Atoi(chunks[3])
	if err != nil {
		return nil, fmt.Errorf("error decoding sysvar::clock timestamp in seconds: %w", err)
	}

	if ctx.slot == nil || uint64(slotNumber) != ctx.slot.Number {
		return nil, fmt.Errorf("read slot %d end not matching ctx slot %s", slotNumber, ctx.slot)
	}

	ctx.slot.TransactionCount = uint32(len(ctx.slot.Transactions))

	if len(ctx.trxMap) != 0 {
		return nil, fmt.Errorf("some transactions are not ended when the slot ends: %q", ctx.trxMap)
	}

	ctx.finalized = true
	return ctx.slot, nil
}

func (ctx *parseCtx) readSlotFailed(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 3 {
		return fmt.Errorf("read slot failed: expected 3 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("read transaction provcess: slotNumber to int: %w", err)
	}

	if ctx.slot == nil || uint64(slotNumber) != ctx.slot.Number {
		return fmt.Errorf("read slot %d failed not matching ctx slot %s", slotNumber, ctx.slot)
	}

	msg := chunks[2]
	ctx.finalized = true
	return fmt.Errorf("slot %d failed: %s", slotNumber, msg)
}

// TRX_S 3XsJkPPXeSCBupg8SyquewZhnDdcch977crSJzXx8NV9SERo9LmUAW36eLokKngzataDvzJ4jwuuW17AkHjpFszu 1 0 3 F8UvVsKnzWyp2nF8aDcqvQ2GVcRpqT91WDsAtvBKCMt9:AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG:SysvarS1otHashes111111111111111111111111111:SysvarC1ock11111111111111111111111111111111:Vote111111111111111111111111111111111111111 7FVmHWPFPxzMK3mHx2y7Q8NG3krPiB142ZG3LZiSkHdX
func (ctx *parseCtx) readTransactionStart(line string) error {
	chunks := strings.Split(line, " ")
	if len(chunks) != 7 {
		return fmt.Errorf("read transaction start: expected 7 fields, got %d", len(chunks))
	}

	sigs := strings.Split(chunks[1], ":")
	id := sigs[0]
	additionalSigs := sigs[1:]

	reqSigs, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("failed decoding num_required_signatures: %w", err)
	}
	roSignedAccts, err := strconv.Atoi(chunks[3])
	if err != nil {
		return fmt.Errorf("failed decoding num_readonly_signed_accounts: %w", err)
	}
	roUnsignedAccts, err := strconv.Atoi(chunks[4])
	if err != nil {
		return fmt.Errorf("failed decoding num_readonly_unsigned_accounts: %w", err)
	}

	accountKeys := strings.Split(chunks[5], ":")

	transaction := &pbcodec.Transaction{
		Id:                   id,
		Index:                ctx.trxIndex,
		AdditionalSignatures: additionalSigs,
		AccountKeys:          accountKeys,
		Header: &pbcodec.MessageHeader{
			NumRequiredSignatures:       uint32(reqSigs),
			NumReadonlySignedAccounts:   uint32(roSignedAccts),
			NumReadonlyUnsignedAccounts: uint32(roUnsignedAccts),
		},
		RecentBlockhash: chunks[6],
		SlotNum:         uint64(ctx.slot.Number),
		SlotHash:        ctx.slot.Id,
	}

	ctx.recordTransaction(transaction)

	return nil
}

func (ctx *parseCtx) recordTransaction(trx *pbcodec.Transaction) {
	ctx.trxMap[trx.Id] = trx
	ctx.trxIndex++
}

func (ctx *parseCtx) readTransactionEnd(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 3 {
		return fmt.Errorf("read transaction start: expected 3 fields, got %d", len(chunks))
	}

	id := chunks[2]
	trx := ctx.trxMap[id]
	ctx.recordTransactionEnd(trx)
	delete(ctx.trxMap, id)

	return nil
}

// TRX_L 2iBoCQ16uhu7ZJdb9icuyS5EpRm6m9RvFH43Co9xi6QkduwE6BtmERtUGwwsutR1tt1L9KfJNi1yNXuy855A4Yan 50726f6772616d20566f746531313131313131313131313131313131313131313131313131313131313131313131313131313120696e766f6b65205b315d
func (ctx *parseCtx) readTransactionLog(line string) error {
	chunks := strings.Split(line, " ")
	if len(chunks) != 3 {
		return fmt.Errorf("read transaction log: expected 3 fields, got %d", len(chunks))
	}

	id := chunks[1]
	trx := ctx.trxMap[id]
	logLine, err := hex.DecodeString(chunks[2])
	if err != nil {
		return fmt.Errorf("log line failed hex decoding: %w", err)
	}
	trx.LogMessages = append(trx.LogMessages, logLine)

	return nil
}

func (ctx *parseCtx) recordTransactionEnd(trx *pbcodec.Transaction) {
	ctx.slot.Transactions = append(ctx.slot.Transactions, trx)
}

// INST_S 27ocnWWBHMWC1ZPfz3kBqyCH1koGsLRNAYe1zp1JhVSeUX3QDniV992yPK7cKFieXViPN9o1bEBEJ55b4wU59WGo 1 0 Vote111111111111111111111111111111111111111 0200000001000000000000000b0000000000000004398c6eecd88cb501e2bd330d15f9810fa76c26f82d165abd0cbb75292ab0e601e64cda5f00000000 Vote111111111111111111111111111111111111111:00;AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG:01;SysvarS1otHashes111111111111111111111111111:00;SysvarC1ock11111111111111111111111111111111:00;F8UvVsKnzWyp2nF8aDcqvQ2GVcRpqT91WDsAtvBKCMt9:11

func (ctx *parseCtx) readInstructionStart(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 7 {
		return fmt.Errorf("read instructionTrace start: expected 7 fields, got %d", len(chunks))
	}
	id := chunks[1]
	ordinal, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("read instructionTrace start: ordinal to int: %w", err)
	}

	parentOrdinal, err := strconv.Atoi(chunks[3])
	if err != nil {
		return fmt.Errorf("read instructionTrace start: parent ordinal to int: %w", err)
	}

	program := chunks[4]
	data := chunks[5]
	hexData, err := hex.DecodeString(data)
	if err != nil {
		return fmt.Errorf("read instructionTrace start: hex decode data: %w", err)
	}

	var accountKeys []string
	accounts := strings.Split(chunks[6], ";")
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

	err = ctx.recordInstruction(trxID, instructionTrace)
	if err != nil {
		return fmt.Errorf("read instructionTrace start: %w", err)
	}

	return nil
}

func (ctx *parseCtx) recordInstruction(trxID string, instruction *pbcodec.Instruction) error {
	trx := ctx.trxMap[trxID]
	if trx == nil {
		return fmt.Errorf("record instruction: transaction trace not found in context: %s", trxID)
	}

	trx.Instructions = append(trx.Instructions, instruction)

	return nil
}

// ACCT_CH 27ocnWWBHMWC1ZPfz3kBqyCH1koGsLRNAYe1zp1JhVSeUX3QDniV992yPK7cKFieXViPN9o1bEBEJ55b4wU59WGo 1 AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG 01000000d1ee412af80c981c82 012333333333323123123123
func (ctx *parseCtx) readAccountChange(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 6 {
		return fmt.Errorf("read account change: expected 6 fields, got %d", len(chunks))
	}
	trxID := chunks[1]
	ordinal, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("read account change: ordinal to int: %w", err)
	}

	pubKey := chunks[3]

	prevData, err := hex.DecodeString(chunks[4])
	if err != nil {
		return fmt.Errorf("read account change: hex decode prev data: %w", err)
	}

	newData, err := hex.DecodeString(chunks[5])
	if err != nil {
		return fmt.Errorf("read account change: hex decode new data: %w", err)
	}

	accountChange := &pbcodec.AccountChange{
		Pubkey:   pubKey,
		PrevData: prevData,
		NewData:  newData,
		NewDataLength: len(newData),
	}

	err = ctx.recordAccountChange(trxID, ordinal, accountChange)
	if err != nil {
		return fmt.Errorf("read account change: %w", err)
	}

	return nil
}

func (ctx *parseCtx) recordAccountChange(trxID string, ordinal int, accountChange *pbcodec.AccountChange) error {
	trx := ctx.trxMap[trxID]
	if trx == nil {
		return fmt.Errorf("record account change: transaction trace not found in context: %s", trxID)
	}

	trx.Instructions[ordinal-1].AccountChanges = append(trx.Instructions[ordinal-1].AccountChanges, accountChange)

	return nil
}

// LAMP_CH 61hY5LpNSSH3zpnxoLYf5pmStN4JRMJ8H4nt4omyNQgaBb78APUetZRw23QdWpZLWF22KG1rBvNdX9XJcut21HQZ 1 11111111111111111111111111111111 499999892500 494999892500
func (ctx *parseCtx) readLamportsChange(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 6 {
		return fmt.Errorf("read lamport change: expected 6 fields, got %d", len(chunks))
	}
	trxID := chunks[1]
	ordinal, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("read lamport change: ordinal to int: %w", err)
	}

	owner := chunks[3]

	prevLamports, err := strconv.Atoi(chunks[4])
	if err != nil {
		return fmt.Errorf("read lamport change: hex decode prev lamports data: %w", err)
	}

	newLamports, err := strconv.Atoi(chunks[5])
	if err != nil {
		return fmt.Errorf("read lamport change: hex decode new lamports data: %w", err)
	}

	balanceChange := &pbcodec.BalanceChange{
		Pubkey:       owner,
		PrevLamports: uint64(prevLamports),
		NewLamports:  uint64(newLamports),
	}

	err = ctx.recordLamportsChange(trxID, ordinal, balanceChange)
	if err != nil {
		return fmt.Errorf("read lamports change: %w", err)
	}

	return nil
}

func (ctx *parseCtx) recordLamportsChange(trxID string, ordinal int, balanceChange *pbcodec.BalanceChange) error {
	trx := ctx.trxMap[trxID]
	if trx == nil {
		return fmt.Errorf("record balanace change: transaction trace not found in context: %s", trxID)
	}

	trx.Instructions[ordinal-1].BalanceChanges = append(trx.Instructions[ordinal-1].BalanceChanges, balanceChange)

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
