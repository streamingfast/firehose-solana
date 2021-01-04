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

type activeSlot struct {
	slot     *pbcodec.Slot
	trxMap   map[string]*pbcodec.Transaction
	trxIndex uint64
}

func newActiveSlot(slot *pbcodec.Slot) *activeSlot {
	return &activeSlot{
		slot:   slot,
		trxMap: map[string]*pbcodec.Transaction{},
	}
}

func (a *activeSlot) recordTransaction(trx *pbcodec.Transaction) {
	a.trxMap[trx.Id] = trx
	a.trxIndex++
}

func (a *activeSlot) recordTransactionEnd(trx *pbcodec.Transaction) {
	a.slot.Transactions = append(a.slot.Transactions, trx)
}

func (a *activeSlot) recordInstruction(trxID string, instruction *pbcodec.Instruction) error {
	trx := a.trxMap[trxID]
	if trx == nil {
		return fmt.Errorf("record instruction: transaction trace not found in context: %s", trxID)
	}

	trx.Instructions = append(trx.Instructions, instruction)
	return nil
}

func (a *activeSlot) recordAccountChange(trxID string, ordinal int, accountChange *pbcodec.AccountChange) error {
	trx := a.trxMap[trxID]
	if trx == nil {
		return fmt.Errorf("record account change: transaction trace not found in context: %s", trxID)
	}

	trx.Instructions[ordinal-1].AccountChanges = append(trx.Instructions[ordinal-1].AccountChanges, accountChange)
	return nil
}

func (a *activeSlot) recordLamportsChange(trxID string, ordinal int, balanceChange *pbcodec.BalanceChange) error {
	trx := a.trxMap[trxID]
	if trx == nil {
		return fmt.Errorf("record balanace change: transaction trace not found in context: %s", trxID)
	}

	trx.Instructions[ordinal-1].BalanceChanges = append(trx.Instructions[ordinal-1].BalanceChanges, balanceChange)

	return nil
}

type parseCtx struct {
	activeSlots       map[uint64]*activeSlot
	lastEndedSlot     uint64
	conversionOptions []conversionOption
}

func (p *parseCtx) getActiveSlot(slotNumber int) *activeSlot {
	if s, found := p.activeSlots[uint64(slotNumber)]; found {
		return s
	}
	return nil
}

func newParseCtx() *parseCtx {
	return &parseCtx{
		activeSlots: map[uint64]*activeSlot{},
	}
}

func (l *ConsoleReader) Read() (out interface{}, err error) {
	ctx := l.ctx
	zlog.Debug("start reading new slot.")
	for line := range l.readBuffer {
		line = line[6:]

		if traceEnabled {
			zlog.Debug("extracting deep mind data from line", zap.String("line", line))
		}

		// Order of conditions is based (approximately) on those that will appear more often
		switch {
		case strings.HasPrefix(line, "SLOT_PROCESS"):
			err = ctx.readSlotProcess(line)

		case strings.HasPrefix(line, "SLOT_END"):
			var slot *pbcodec.Slot
			slot, err = ctx.readSlotEnd(line)
			if slot == nil && err == nil {
				// We just read a slot_end that was not continuous based on the last seen slot num before it, skipping it
				continue
			}

			if slot != nil && err == nil {
				// We read slot_end that is correct, return it to reader of "blocks"
				return slot, nil
			}

			// All other cases means err != nil, let it be handled at the end of switch

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

func (ctx *parseCtx) readSlotProcess(line string) error {
	zlog.Debug("reading slot process", zap.String("line", line))

	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 16 {
		return fmt.Errorf("expected 16 fields got %d", len(chunks))
	}

	isFull := chunks[1] == "full"
	slotID := chunks[3]
	slotPreviousID := chunks[4]

	slotNumber, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	//if ctx.lastEndedSlot != 0 && uint64(slotNumber) < ctx.lastEndedSlot {
	//	zlog.Warn("skipping slot process not greater then last seen slot",
	//		zap.Int("received_slot_num", slotNumber),
	//		zap.Uint64("last_ended_slot_num", ctx.lastEndedSlot),
	//		zap.String("line", line),
	//	)
	//	return nil
	//}

	rootSlotNum, err := strconv.Atoi(chunks[8])
	if err != nil {
		return fmt.Errorf("root slot num to int: %w", err)
	}

	var activeSlot *activeSlot
	if activeSlot = ctx.getActiveSlot(slotNumber); activeSlot == nil {
		activeSlot = newActiveSlot(&pbcodec.Slot{
			Version:     1,
			Number:      uint64(slotNumber),
			PreviousId:  slotPreviousID, //from fist full or partial
			Block:       nil,
			RootSlotNum: uint64(rootSlotNum),
		})
		ctx.activeSlots[uint64(slotNumber)] = activeSlot
	}

	// We check after the other conditions above to ensure we do not check the map
	// when receiving some out of order SLOT_PROCESS message
	if len(activeSlot.trxMap) != 0 {
		return fmt.Errorf("all transactions should have ended when processing SLOT_PROCESS line: %q", activeSlot.trxMap)
	}

	if isFull {
		activeSlot.slot.Id = slotID
	}

	return nil
}

// SLOT_END SLOT_NUM GENESIS_UNIX_TIMESTAMP CLOCK_UNIX_TIMESTAMP
func (ctx *parseCtx) readSlotEnd(line string) (*pbcodec.Slot, error) {
	zlog.Debug("reading slot end", zap.String("line", line))

	if len(ctx.activeSlots) == 0 {
		return nil, fmt.Errorf("received slot end while no slot is active in context")
	}

	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 4 {
		return nil, fmt.Errorf("expected 4 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return nil, fmt.Errorf("slotNumber to int: %w", err)
	}

	activeSlot := ctx.getActiveSlot(slotNumber)
	if activeSlot == nil {
		return nil, fmt.Errorf("slot end: received slot num (%d) not matching any active slot number in context", slotNumber)
	}

	// We check after the other conditions above to ensure we do not check the map when receiving some out of order SLOT_END message
	if len(activeSlot.trxMap) != 0 {
		return nil, fmt.Errorf("some transactions are not ended when the slot (%d) ends: %q", slotNumber, activeSlot.trxMap)
	}

	slot := activeSlot.slot
	genesisTimestamp, err := strconv.Atoi(chunks[2])
	if err != nil {
		return nil, fmt.Errorf("error decoding genesis timestamp in seconds: %w", err)
	}
	activeSlot.slot.GenesisUnixTimestamp = uint64(genesisTimestamp)

	clockTimestamp, err := strconv.Atoi(chunks[3])
	if err != nil {
		return nil, fmt.Errorf("error decoding sysvar::clock timestamp in seconds: %w", err)
	}

	slot.ClockUnixTimestamp = uint64(clockTimestamp)
	slot.TransactionCount = uint32(len(activeSlot.slot.Transactions))

	delete(ctx.activeSlots, slot.Number)
	ctx.lastEndedSlot = slot.Number

	return slot, nil
}

// SLOT_FAILED SLOT_NUM REASON
func (ctx *parseCtx) readSlotFailed(line string) error {
	zlog.Debug("reading slot failed", zap.String("line", line))

	if len(ctx.activeSlots) == 0 {
		return fmt.Errorf("received slot failed while no slot is active in context")
	}

	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 3 {
		return fmt.Errorf("read slot failed: expected 3 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	activeSlot := ctx.getActiveSlot(slotNumber)
	if activeSlot == nil {
		return fmt.Errorf("slot failed: received slot num (%d) not matching any active slot number in context", slotNumber)
	}

	return fmt.Errorf("slot %d failed: %s", slotNumber, chunks[2])
}

// TRX_START SLOT_NUM SIG1:SIG2:SIG3 NUM_REQUIRED_SIGN NUM_READONLY_SIGN_ACT NUM_READONLY_UNSIGNED_ACT ACTKEY1:ACTKEY2:ACTKEY3 RECENT_BLOCKHASH
func (ctx *parseCtx) readTransactionStart(line string) error {
	chunks := strings.Split(line, " ")
	if len(chunks) != 8 {
		return fmt.Errorf("read transaction start: expected 8 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	activeSlot := ctx.getActiveSlot(slotNumber)
	if activeSlot == nil {
		return fmt.Errorf("transaction start: received slot num (%d) not matching any active slot number in context", slotNumber)
	}

	sigs := strings.Split(chunks[2], ":")
	id := sigs[0]
	additionalSigs := sigs[1:]

	reqSigs, err := strconv.Atoi(chunks[3])
	if err != nil {
		return fmt.Errorf("failed decoding num_required_signatures: %w", err)
	}
	roSignedAccts, err := strconv.Atoi(chunks[4])
	if err != nil {
		return fmt.Errorf("failed decoding num_readonly_signed_accounts: %w", err)
	}
	roUnsignedAccts, err := strconv.Atoi(chunks[5])
	if err != nil {
		return fmt.Errorf("failed decoding num_readonly_unsigned_accounts: %w", err)
	}

	accountKeys := strings.Split(chunks[6], ":")
	recentBlockHash := chunks[7]

	transaction := &pbcodec.Transaction{
		Id:                   id,
		Index:                activeSlot.trxIndex,
		AdditionalSignatures: additionalSigs,
		AccountKeys:          accountKeys,
		Header: &pbcodec.MessageHeader{
			NumRequiredSignatures:       uint32(reqSigs),
			NumReadonlySignedAccounts:   uint32(roSignedAccts),
			NumReadonlyUnsignedAccounts: uint32(roUnsignedAccts),
		},
		RecentBlockhash: recentBlockHash,
		SlotNum:         uint64(activeSlot.slot.Number),
		SlotHash:        activeSlot.slot.Id,
	}

	activeSlot.recordTransaction(transaction)

	return nil
}

// TRX_END SLOT_NUM TX_SIGNATURE
func (ctx *parseCtx) readTransactionEnd(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 3 {
		return fmt.Errorf("read transaction start: expected 2 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	activeSlot := ctx.getActiveSlot(slotNumber)
	if activeSlot == nil {
		return fmt.Errorf("transaction end: received slot num (%d) not matching any active slot number in context", slotNumber)
	}

	id := chunks[2]
	trx := activeSlot.trxMap[id]
	activeSlot.recordTransactionEnd(trx)
	delete(activeSlot.trxMap, id)

	return nil
}

// TRX_L SLOT_NUM TX_SIGNATURE LOG_IN_HEX
func (ctx *parseCtx) readTransactionLog(line string) error {
	chunks := strings.Split(line, " ")
	if len(chunks) != 4 {
		return fmt.Errorf("read transaction log: expected 4 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	activeSlot := ctx.getActiveSlot(slotNumber)
	if activeSlot == nil {
		return fmt.Errorf("transaction end: received slot num (%d) not matching any active slot number in context", slotNumber)
	}

	id := chunks[2]
	trx := activeSlot.trxMap[id]
	logLine, err := hex.DecodeString(chunks[3])
	if err != nil {
		return fmt.Errorf("log line failed hex decoding: %w", err)
	}
	trx.LogMessages = append(trx.LogMessages, string(logLine))

	return nil
}

// INST_S 10 aaa 1 0 Vote111111111111111111111111111111111111111 0200000001000000000000000b0000000000000004398c6eecd88cb501e2bd330d15f9810fa76c26f82d165abd0cbb75292ab0e601e64cda5f00000000 Vote111111111111111111111111111111111111111:00;AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG:01;SysvarS1otHashes111111111111111111111111111:00;SysvarC1ock11111111111111111111111111111111:00;F8UvVsKnzWyp2nF8aDcqvQ2GVcRpqT91WDsAtvBKCMt9:11
func (ctx *parseCtx) readInstructionStart(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 8 {
		return fmt.Errorf("read instructionTrace start: expected 8 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	activeSlot := ctx.getActiveSlot(slotNumber)
	if activeSlot == nil {
		return fmt.Errorf("instruction start: received slot num (%d) not matching any active slot number in context", slotNumber)
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

	err = activeSlot.recordInstruction(id, instruction)
	if err != nil {
		return fmt.Errorf("read instructionTrace start: %w", err)
	}

	return nil
}

// ACCT_CH 10 aaa 1 AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG 01000000d1ee412af80c981c82 012333333333323123123123
func (ctx *parseCtx) readAccountChange(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 7 {
		return fmt.Errorf("read account change: expected 7 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	activeSlot := ctx.getActiveSlot(slotNumber)
	if activeSlot == nil {
		return fmt.Errorf("account change: received slot num (%d) not matching any active slot number in context", slotNumber)
	}

	trxID := chunks[2]
	ordinal, err := strconv.Atoi(chunks[3])
	if err != nil {
		return fmt.Errorf("read account change: ordinal to int: %w", err)
	}

	pubKey := chunks[4]

	prevData, err := hex.DecodeString(chunks[5])
	if err != nil {
		return fmt.Errorf("read account change: hex decode prev data: %w", err)
	}

	newData, err := hex.DecodeString(chunks[6])
	if err != nil {
		return fmt.Errorf("read account change: hex decode new data: %w", err)
	}

	accountChange := &pbcodec.AccountChange{
		Pubkey:        pubKey,
		PrevData:      prevData,
		NewData:       newData,
		NewDataLength: uint64(len(newData)),
	}

	err = activeSlot.recordAccountChange(trxID, ordinal, accountChange)
	if err != nil {
		return fmt.Errorf("read account change: %w", err)
	}

	return nil
}

// LAMP_CH 10 aaa 61hY5LpNSSH3zpnxoLYf5pmStN4JRMJ8H4nt4omyNQgaBb78APUetZRw23QdWpZLWF22KG1rBvNdX9XJcut21HQZ 1 11111111111111111111111111111111 499999892500 494999892500
func (ctx *parseCtx) readLamportsChange(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 7 {
		return fmt.Errorf("read lamport change: expected 7 fields, got %d", len(chunks))
	}

	slotNumber, err := strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("slot num to int: %w", err)
	}

	activeSlot := ctx.getActiveSlot(slotNumber)
	if activeSlot == nil {
		return fmt.Errorf("account change: received slot num (%d) not matching any active slot number in context", slotNumber)
	}

	trxID := chunks[2]
	ordinal, err := strconv.Atoi(chunks[3])
	if err != nil {
		return fmt.Errorf("read lamport change: ordinal to int: %w", err)
	}

	owner := chunks[4]

	prevLamports, err := strconv.Atoi(chunks[5])
	if err != nil {
		return fmt.Errorf("read lamport change: hex decode prev lamports data: %w", err)
	}

	newLamports, err := strconv.Atoi(chunks[6])
	if err != nil {
		return fmt.Errorf("read lamport change: hex decode new lamports data: %w", err)
	}

	balanceChange := &pbcodec.BalanceChange{
		Pubkey:       owner,
		PrevLamports: uint64(prevLamports),
		NewLamports:  uint64(newLamports),
	}

	err = activeSlot.recordLamportsChange(trxID, ordinal, balanceChange)
	if err != nil {
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
