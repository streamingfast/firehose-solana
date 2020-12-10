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

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"go.uber.org/zap"
)

var supportedVersions = []string{"12", "13"}

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
	slot          *pbcodec.Slot
	activeSlotNum uint64
	trxIndex      uint64
	trx           *pbcodec.Transaction
	trxTrace      *pbcodec.TransactionTrace

	conversionOptions []conversionOption
}

func newParseCtx() *parseCtx {
	return &parseCtx{
		slot:     &pbcodec.Slot{},
		trx:      &pbcodec.Transaction{},
		trxTrace: &pbcodec.TransactionTrace{},
	}
}

func (l *ConsoleReader) Read() (out interface{}, err error) {
	ctx := l.ctx

	for line := range l.readBuffer {
		line = line[6:]

		if traceEnabled {
			zlog.Debug("extracing deep mind data from line", zap.String("line", line))
		}

		// Order of conditions is based (approximately) on those that will appear more often
		switch {
		case strings.HasPrefix(line, "TRANSACTION"):
			err = ctx.readTransactionStart(line)

		default:
			zlog.Info("unknown log line", zap.String("line", line))
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

type creationOp struct {
	kind        string // ROOT, NOTIFY, CFA_INLINE, INLINE
	actionIndex int
}

func (ctx *parseCtx) resetBlock() {
	if ctx.activeSlotNum != 0 {
		ctx.resetTrx()
	}

	ctx.slot = &pbcodec.Slot{}
}

func (ctx *parseCtx) resetTrx() {
	ctx.trxTrace = &pbcodec.TransactionTrace{}
	ctx.trx = &pbcodec.Transaction{}

}

func (ctx *parseCtx) readSlotStart(line string) error {
	ctx.resetTrx()
	ctx.activeSlotNum = 0 //todo: get slot from line ...
	return nil
}

func (ctx *parseCtx) readTransactionStart(line string) error {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) != 5 {
		return fmt.Errorf("read transaction start: expected 5 fields, got %d", len(chunks))
	}

	ctx.resetTrx()

	id := chunks[3]
	signatures := []string{id}

	solMessage := &solana.Message{}
	messageData, err := hex.DecodeString(chunks[4])
	if err != nil {
		return fmt.Errorf("read transaction start: hex decode message: %w", err)
	}

	err = bin.NewDecoder(messageData).Decode(&messageData)
	if err != nil {
		return fmt.Errorf("read transaction start: binary decode message: %w", err)
	}

	var accountKeys [][]byte
	for _, k := range solMessage.AccountKeys {
		accountKeys = append(accountKeys, k[:])
	}

	var instructions []*pbcodec.CompiledInstruction
	for _, i := range solMessage.Instructions {

		var accountIdIndexes []uint32
		for _, i := range i.Accounts {
			accountIdIndexes = append(accountIdIndexes, uint32(i))
		}

		instruction := &pbcodec.CompiledInstruction{
			ProgramIdIndex: uint32(i.ProgramIDIndex),
			Accounts:       accountIdIndexes,
			Data:           i.Data,
		}
		instructions = append(instructions, instruction)
	}

	message := &pbcodec.Message{
		Header: &pbcodec.MessageHeader{
			NumRequiredSignatures:       uint32(solMessage.Header.NumRequiredSignatures),
			NumReadonlySignedAccounts:   uint32(solMessage.Header.NumReadonlySignedAccounts),
			NumReadonlyUnsignedAccounts: uint32(solMessage.Header.NumReadonlyUnsignedAccounts),
		},
		AccountKeys:     accountKeys,
		RecentBlockhash: solMessage.RecentBlockhash[:],
		Instructions:    instructions,
	}

	transaction := &pbcodec.Transaction{
		Id:         id,
		Index:      ctx.trxIndex,
		Signatures: signatures,
		Msg:        message,
	}
	ctx.recordTransaction(transaction)

	transactionTrace := &pbcodec.TransactionTrace{
		Id:       id,
		Index:    ctx.trxIndex,
		SlotNum:  uint64(ctx.slot.Number),
		SlotHash: ctx.slot.Id,
	}

	ctx.recordTransactionTrace(transactionTrace)
	return nil
}

func (ctx *parseCtx) recordTransaction(trx *pbcodec.Transaction) {
	ctx.slot.Transactions = append(ctx.slot.Transactions, trx)
}

func (ctx *parseCtx) recordTransactionTrace(trxTrace *pbcodec.TransactionTrace) {
	ctx.trxTrace = trxTrace
	ctx.trxIndex++

	return
}

func splitNToM(line string, min, max int) ([]string, error) {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) < min || len(chunks) > max {
		return nil, fmt.Errorf("expected between %d to %d fields (inclusively), got %d", min, max, len(chunks))
	}

	return chunks, nil
}
