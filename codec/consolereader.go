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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/mr-tron/base58"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"github.com/streamingfast/solana-go"
	"go.uber.org/zap"
)

var MaxTokenSize uint64
var VoteProgramID = solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111111")

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

	zlog.Info("done scanning lines")
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
	case b := <-ctx.blockBuffer:
		return b, nil
	default:
	}

	for line := range r.lines {
		//fmt.Println("processing lines")
		if !strings.HasPrefix(line, "DMLOG ") {
			zlog.Debug("node", zap.String("log", line))
			fmt.Println("node:", line)
			continue
		}

		line = line[6:] // removes the DMLOG prefix
		if err = parseLine(ctx, line); err != nil {
			return nil, r.formatError(line, err)
		}

		select {
		case b := <-ctx.blockBuffer:
			return b, nil
		default:
		}
	}

	select {
	case b := <-ctx.blockBuffer:
		return b, nil
	default:
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
		//err = ctx.readBlockRoot(line)

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
			if bytes.Equal(i.ProgramId, VoteProgramID[:]) {
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
	BlockWorkChunkSize   = 15
	BlockEndChunkSize    = 5
	BlockFailedChunkSize = 3
	BlockRootChunkSize   = 2
	InitChunkSize        = 3
	SlotBoundChunkSize   = 3
)

type bank struct {
	parentSlotNum   uint64
	processTrxCount uint64
	previousSlotID  []byte
	transactionIDs  [][]byte
	blk             *pbcodec.Block
	ended           bool
	batchAggregator [][]*pbcodec.Transaction
}

func newBank(blockNum, parentBlockNumber uint64, previousSlotID []byte) *bank {
	return &bank{
		parentSlotNum:   parentBlockNumber,
		previousSlotID:  previousSlotID,
		transactionIDs:  nil,
		batchAggregator: nil,
		blk: &pbcodec.Block{
			Version:       1,
			Number:        blockNum,
			PreviousId:    previousSlotID,
			PreviousBlock: parentBlockNumber,
		},
	}
}

// the goal is to sort the batches based on the first transaction id of each batch
func (b *bank) processBatchAggregation() error {
	indexMap := map[string]int{}
	for idx, trxID := range b.transactionIDs {
		indexMap[string(trxID)] = idx
	}

	b.blk.Transactions = make([]*pbcodec.Transaction, len(b.transactionIDs))
	b.blk.TransactionCount = uint32(len(b.transactionIDs))

	var count int
	for _, transactions := range b.batchAggregator {
		for _, trx := range transactions {
			trxIndex := indexMap[string(trx.Id)]
			trx.Index = uint64(trxIndex)
			count++
			b.blk.Transactions[trxIndex] = trx
		}
	}

	b.batchAggregator = nil

	if count != len(b.transactionIDs) {
		return fmt.Errorf("transaction ids received on BLOCK_WORK did not match the number of transactions collection from batch executions, counted %d execution, expected %d from ids", count, len(b.transactionIDs))
	}

	return nil
}

func (b *bank) getActiveTransaction(batchNum uint64, trxID []byte) (*pbcodec.Transaction, error) {
	length := len(b.batchAggregator[batchNum])
	if length == 0 {
		return nil, fmt.Errorf("unable to retrieve transaction trace on an empty batch")
	}
	trx := b.batchAggregator[batchNum][length-1]
	if !bytes.Equal(trx.Id, trxID) {
		return nil, fmt.Errorf("transaction trace ID doesn't match expected value: %s", base58.Encode(trxID))
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

// BLOCK_WORK 10822740 10822740 full 4TbxQJpq7MT843rFzLwR3yLKcsRiLi7E5TQi3TD8LR5A 974 59s 89s 3 0 0 0 4TbxQJpq7MT843rFzLwR3yLKcsRiLi7E5TQi3TD8LR5A 0 T;668tfTrSGexUzimCftbizyuYqSeoEQPuE6QRmz6XCddtCxhBZm36Eaeid7EaDGCeMHXDenBHFKVRr7Djgzvk82hf;4xdihXZj7c9xTCTSS18PJZuSZ6GzzsxfZBqR2xiYU1XzSjx9wYxPoN4QQoqDUckSDVLVh5PzD7WUoxHfjnUqQ44r;5ZqiuFuyszM2G535noYpjHN9k6GQ2SuwVzAkQdXYx5uW9bFB5ujByaVTpKqDixbQmpxiRC4EuBBphKByKrYhFx9i;2gzbmbhUPSvV3H4EmG5E8MgKbiNRfcRoTVX7n7yqiB1Evhgpw5xHvk2KasRJ4hDfE3dhdzA2CPWcqLkyKmqxxH8d;5ZofEU5Hx5L7yTiRNAqgHo2e6R15BHvg1nXjnP9jyYJHL42BPw9ZwaAfmif4WwPxzQnix62JhXHABZYqLkeeNcr5;63zmRVeVt6ooveRpixkMphj2rMAMuZujMccSHwNe3iaJRDogtsZdpv5SFiaJnV8mkZzowHsGgLCunkcUnZ6KPPDE;2uxnJAJ8cxb3FcLXH92ezHfh6T2Row7QYtTF8Y5E2m4YpcUZDriYRnHixtDYdC8bvkf51zccRjCsv18RFdiVhUtL;5BwmHtuPcivoRK4bX91rmf9N7uNAnwu1WTki4zxezrb7NhLWPBywLRsAbp845TXZs2jPsjehWx7MGLPaFLK6mtta;37rnA3CEpdtJXR8YndYMAYVBwBw9UXAdRZQtd8Z3zzMwbNoofLLMetRTJUTLGWBGx6vhSw5WQyuesgzAaHdaRH5L;wJ6fDFUrQ1gDrpqpykBdswq7etvfaT7Y93ja9v3szuCAKsSDNqV8UCjk4zdhiqzqotqLUKwSqGPUnkXHUCbeWiX;MYvJEmKDyfbY6ZixpSAvXur3jJ3eA18K4U717CTWFVh45bs51r5myyUh1sbSNxRuEgLp8Z6iCzZggddtK8WCRDG;JWRSzMnHWFAUU8qbp2vMYvwEhv2dyxzEstXEAxFgC49WDr3kjD6MryvMzMnTyvyWrEzdwud2tnFewYrPAgAQNrk;2TvPWJ7RFiJUHYSEgHEMueDBd3Fj1gLpuFC5PF21wpo9uyyeuj3t57DUJsoAvSfP2PtynC6o8suppDhNc1D5KYH2;5zYKfMR8bLd9ZgFSLWDRiw2A69oEZtxYic6aeHkHSoJPi63YpyA9aWAqkozNyk8de3eEBtxPvAJjMMEqu9xThgwN;2SGyVco2nnr2Hz2uaV9bZYx1ckLHueKUDm1tcF7ssStvbcTaTbJjDrgEEzjudrS2Lcp1jpDAre82imUvkYYzdkKL;2PJHpZy2mnzFWGTf9QwiU8pciZVGjHun8jankMocY8DReFnMeLMLw6qX2JPWcDLc8fNrZMhiQQTMVaErdop2iPxX;3sngyaPw8GFM5Yhx76iEDt3bxNkn4Yn4vfzoxy8sr4mzbCZRtqtsybyrfPWFDjQCAmAwpv9ScLpZKrmBh5wNtgRL;3ecpQQUu3k236mbtKfUrBBVV6vaigzjBn8wYVhNkVSKRaYz9wErHoKMDk3LKzr3D6oh7TZxQx1FUGt8Ya7arGYCj;2k8d1bm1DWkWd9NWNkVUvRcccpWDmWm4pQLNyCoiSDPupMbUuYR7ddz9MCjF29GWnmpNYqPQKhDJkkXxfo2o4tsB;2LoPf3n8r2TBvxQUrQZNxqhAe5Dc2kV2fUXZd3dSxyX71zRg6GTQjzJmByLeWRDyZDwpoLJXhr6ZgEE66D3VFrBu;5eFxbm1trKtowgGTzoQzgmvKGWqTAviKgBeVoN8QYp88k56GfhvxzFtgdgrbvffvdkPTvgiW8DtU2ZQL7H2oc9Dv;2WXhithCc7pdnpZMEj5UFubkAzKFiT4PLHmtFLeEwDWJtyzXLs3dpQGvofWGLEsqQnWL4iJeeFmf9LXaMAd2dor8;5xaWH3WRMPEvC3Yhdghtyte8EgxBo8eD8tGKSNS4A8dsqydAB6cQBfiywvUZ1pjmyu6Ho6w8YLcoEvLb5v1wPUvk;4L3kzmzmJzd18x8H6x4AQWM2e8tnFDLUEaGupVzR55fspCbNASWoSVrHT3xPafXNfpgU7RqrztBNE9UMufB94ay1;PvR74QUmQk4MitsAEtutHjkf68E8dFYWq2nNcHyKisvCQKzJFdDFCMSriX2LtcfTVNtJB7c669UdU2FAi6s54m5;4hf8yVjf8yZ1ps31LEpwWcFUXoDb3aSpXkSnLRu7ZVg37w6s1kTM7mb1byJ1edsaNymGFkdoZYFfVxfGNuNThsGT;2iSUv5Rp6XzJCz7vDUtiULyv22wgjekYekKQtY6hcYuJ9c2eDV3jpUepJNQvoYZvg2xHSbcoWpdfx68KMcok9PRM;2GLZPb8scxqJSkHwC9nsaA1aZa968X7W5o9reZ5b8DYNiYgsUSXDwgxLARMYxrU3a6humpUiKK6X8wdHg5pMHP6v;GfbjFvP4Uvy2YVKpmaUaqa717CuieqXmyEbMP4TDW42oQQ3kpuEicCRrGVDhecuVqB7bZqXtZBLiTCti2gihDG2;a9yXmpJWnaSxJ3bR8NvhAc9yu1BTFYabgBAw723WoQhnCnPP1cPoMuvSXfW5ajvKAbz9PTVMdEfvJqGY1sz3TbD;7eXbdfsiEEEcX5EpwwExJiroHtumr7P53b1aVaEDQGzd2m33VNQJrYiWkdcKDYFKxd3zTykwACWFkdA8WzyNwnS;22o6WQSYmDaXRLoe8JdQyUhr4hv1FStoyiV1LDnRu4WaS1iVZduqiLaDZD8KS5T5Ccjd8wB4yP7NVkDAdASWtPi4;2XSazm3XNTfSH6Vtg3DGUvu3iTLa4zKpocLQ3MU4eydC2mN3qXh5gGM7xYcSYcfeDXSttMMa8bMPrjxEDBkGdUro;4K92S9ZJNvgAdkgwtuR8qUXSB9pLEozYGPPoiFJ1QVyXn7KCwp9usf5JrBRxh7BREZ8xgK8nDUFkb5ZMUHzv5xdG;5zXKFY9jW8VuzhS8RSLF7i9kusVAKZ9zrY5eEzhTpssUNHFpF5aCMnwFRgRpaEWp2TKwYFKBggvRYkQDyNuZ8tA1;4ySDmzYGCYProtCphszeuTK2PcjbVM9mZdYTjmUQ8sS3BN7VYECofDSv5sdmJdMFSmof34MdWimFxo1RJDBHZCpL;4rMZpXD6rfdMHKX79Z3LfKVZM23GS7QYNzYzHBxzL9qKGxfpX26Hp3mQBQNbtt3EKTbD7MAbKGM3qu6gowPdQHap;2Y7zNTwoGXQVTzkXNpwVR7LRGcpbfBDZHHqj5GEwBf8Q4zBTqHH25SoVGZJui18kkaYtLgwQXnuNTTXmXAkMdiUY;3pH69afRwXJF1kSMne7xbyoAmkibr3THLBavJaRUmMKwRWwvekPoLJLk5d2PuCJ2eW9ickfuCN2ytdrdnaPiMjkr;34M2hKQnZQgvzmak6rLNeHkGQbEp4WqbU9txhWcyMRqz6PBnuBTLjr1zJ4ee1uWLZAFmbhwb8mwcTNgdzs5JuHAN;57wbxKmjTa9DUiDHU77uwmEwXYDPG1nbDF14LM2q2bFXftV4LKTBSNGmFx5kacYjktkbNHFZwb2FQsDXxLd9jrZL;5rmSYiKYFoqYovrrgigtrhkkwz478yS2KTs9RePP8z2v9Jjek2JNcin2HdQAzt2RwAE6Ni2ATVdhCRHFod4zaE8C;22e8e3Gd9wzyGpK8oTe72jjHr5uN1eH6pgYa5ZkwEzA4dA1hVuBt6vYKb4MCQnVKYcSxyaJdWRTws1uWbZdnA1Kv;3TDJj3TNeBtmwCBmVZkAWnzMYCugEDgm9vAQhyz6aRM2HpeDqjCBUQ9VQN2qCuEZaMCg8FTjNpZ1JiNXvPJdUFTR;GKuJJUxGeSFA9xfhrJLkAaUgvGjxetsiLLCXaTTHpXutoKnHM15kA2h8B9YeyU7ZUw6pnFtAUTx6MtrdPAa1zpt;4psBf8sJMzAsPpKaDNfuaY1BRykSE7T9rArkQnHfq4cc8PTcJWpr3f2kzjgjzzZmmgjoKp9eYdcovzuyqjueCwLY;3idfGKuGvNsL4btjwZMxrQUs9guozqkWNW1Jfp1hQ49UJupudnQuC54SZLXr5ueBLEsQ4R5rLeLdo65TGM6Upqor;3s8UFvWwCTvVdp7JouFoximWTnVUKyZJRS7DWbDzANS4AsB3r1N8FHY4uYxHVj6q3RhTwTfDzaqmkbNDj2voJPXu;mojuR9vCpa8BTuVKw72tQzA2KSnkBtcLjBTSTwtpW4sAneDodAmBpDUodZHm7n19DKhCCyTxmX2SHTGMwZtrwYB;35VxMkqAe56EVpQXXvpK4BLnT6aAc3UH6Xx3jo4PVtoeZwveHXtQaV8KZkdtafpHVdvD9WBzX9Qs3TBwhxsAiSM9;uC9RuAqagQzb244bMej8sJYbmEkJP4Mqm1vaJBFsm1qGqmy2Ek3x8gmhaCyxZkwUKJsKSqpmghBUSwnfc5uocLR;2bzVNGBmAXDGMNUbeytzvAJY9EtxaxK8jDmssnev7EVuGk9LXuVh67uQV8rEcqL2ZfySCgPNQ76ULULWvYYheJSR;5z711SXSiP2RydvcGy9mvvDU6ceRJgecaEVbENC8PGeWuuTbGgDpXmJkMwDftnHttYJzpWgyxXsYpGyRGowmdcXt;4UWUzU3Cp3x5xCNWKLA3unMqcFvF2NnghzHu7CwnEEZVmgcThSQw4LjqDhjKx5WvBbks5VqNRq5E6ANwbMgrCard;Lve7sFokh224HDNEYZqf1TntagRA18JmdeNnF6uoNve1EnFTSHZ4DtWVo4sUjSXaaCFtCDU8QXu8jsdLuyXY1Sn;59aUNSMsLePvpxf7DwAJgUM7ZQ8UEp1FdVrxF8BicQFisQmH1x7zBTKReYtvAUyFmGUmfdqsP8631em5dk7H1bHZ;3Ns2fW5216CebGthy1TfNXeRYb3GwRVo3rdawe3nAxBx6rRxfKeBkv6iHDBJA64NcsbuqxgtdgGTqEyd85p9tcu1;5SWGqkYpJ3S9FbV3DKUt7faB1u6SutVYJZkZj68jCun4MVGEHhEH4J1wvLN1CF4tV6HzzgvGXFLP1S2mZCQbfTPw;2UckTkgm7JM4vimoRJasQpqfL6naNqcLGryaHxAFSfAtVug89uk7k5VYSjvCmh3MC9aWHRB2TKwkvzn82YYrxZ6a;4VSrUWuNJv9cTkAXiJnQQwSF7tpadPsLjx89wyGJnBU27rtxxKkakVDwUw6VuWTi6Ni4Vtz3SqAUGiWSjnQgE28y;38gbDWPHtNvQMm9VgpJoRDZpgHtQiYz8Po54cbuxj7eHHmPexasN6sfZ9eepieHmscVkGbbLgqjYC3iA99LQfTcs;2HXTPKGLqTxyYxxAvbHCMyuSA8XzNotMP4LAf9a1UabfkAohLGZG8GYNx5h8uh2Sv7oJNmtumsNRTXsERNzJxPG4;28xgBh1g2LPsnfassTEd2fEZHXrzRmvJhrT95eVXfx6QYu33KkgVt82nKQR3uWEyBfaZsUZsBPzPcpv5QspSobhB;4ZktRiC9eDRv7W9RduSL1tZRMyK6H1DG59a2n8Lgps6vfzM2hEsLLX8x28iqgAz4v8Bj3nC11CMxGJdFR1uY2hDS;5woi4YbMmgxyjTa4b2XsbHj9ByVAiuQ4y4rAAoTczhx2Wjti1FUUymD5GMaENGnpLYkyZZXmM98Bd3D7oYjmAAew;4Voct1cDCnwiKq5TLfrrTKFMHmdMmeujtGdufD9Kjyx3JwHrxd6u7u6toB5DbEysvisMX5GSpPD4vea5TcaQbmtm;39c23pbstSN1fwmkbD63iV3yLew18RDi2JQtaxbkcCynFYQKa1TsanuaEfCmwGf7iq9SugBpPoXabT6odBtZ6U1S;CrLowTHETYoWpLmnJAiyhSj2sbHjKfbY6xFkaR1YxGaUGmSwzMPGM9fZj8Qs7ypES1Q629NcHRbb1Hp8sR5CRxu;59htB9qCAKfiwPghH8YzpT31xAp2C81xiGD2tWFBGGnGccwH8JeTsbpJxDzBXhbjmYhPQp2Hx6SjE6iv4zREJjuT;3kMv1u77kvZMYKcbN2EMNd9zoQYFKMKrRm2FoupkaPFydRYt47g3CYZPyfyPDDXeWwxGZuSrdrqRqL8xUV5Fon3o;2WqUxH9ikrhidxGPjp7bpHcHknfFuYdTjEB4RR5uicicZ9ZJuhxV67vLNz2s1ZYspZ46JvdwLsPJxTEZhiXCfDMi;3PsKf9o6LE2TzNrGzZBbLtB3aVA8UpsJpLx8v1D4LcTVb3tQfaR2GWx2UFjYg4iunPYeK7qz3LYSFzxQ5eye6z45;2tWGEUB8YrPztJy41ekLmbYwxsgaKKcEYRUT2fWqFM6zJ129t9qyk6UNyECT5PJPgnJ4p1Ji65ceqhBjfj4z8e94;2J5GE1agjhnJ1sUaahDXTgA8V9omAF3bHr2sWBe6NLEg7CwECspG1SfCx1y9DfLDZXbAqCNPjBwauvMM2VPDM98n;4EiDavNDQY683YKnpDtfX3kvSnLHpVz7Fa913NcXNJy3LsoDd8CNHT44RiE1XWRfDqsDatru3uRp2meDfmtEfDCw;2ZUXGnJhDa9PVsWyNkDh4NzgTd2J4xewgwMEDgawXjfdo894ZEMdVFUih8HCCP7HB5P4qra2PwyZkdaoQo7f6A4m;2Lo9KGbEtahtSj3jLnzubUH831PFr68YqQjQyxQS5Mm5qmRu78qmEA4HjbGYGntpgydJp8QpjL4Entr2WXpL8Yqc;5LB3bKA9gBzu9SQEzPSo1WZRqZGbtonm5nsGuZuihnLigwKfR3S9bbrjStxpfCEHZZa6mdat5Lz2vbMzG952Y34h;Bp6PAntbMEaippQXFqeDzBq3RwWFYf1xmX56xti5g8DTGWGbhZqCADH2rGNQMFwEni3bkFGYiRUbuMEaUFPzPov;3mSKaXvj7YNuLiawdQZ2XNnCDqa5noMPju6m2Z2Thv171ad6omMprtsNBHAUFRn4ay5EMscWBfp3CRosys2UBHwq;4vPURw2Kd9g3rjuBTpv7t2T71d9HuLFxWwFCuUabwu8xu7GwQAv3UhWt1jVnHj4DPs19V9teve7rmMc8hHTTeFVP;2XbRwK5yV2wK1RDSafoxVUiyx1y1YCzGFSDr2ccixsUb8ds9RkUvx735PJHKgt5QMEyjacCYQz1UAvgrmpLa13PH;41yBCYD5n2CDtHpcMAzmefKfVNL3kBXRz1RLGUQmat2GzKVsjsDV7QtKeeUbzcdt68WhxVKLkbhKvavhfzeH9F7J;4Fs7WatksKuKLovvth4T5mirhCEpLwSLBuFABnQKBKhqfhU3jfZHhpuW1DW3Lsxa83wWPb76GipdS26XyV5RA2x5;avHQMQLAc2KUEQ2VYJfzHsk23zbHqjYLkcLM2EbmJpyBZkPsyRE6xfYzLnTJAEYJnB39AgU5fTvasxWH6VKy3mm;4TnP72QHzk5asFDq7n6K1sdUTnNseiqgypYoATBSGuLQaaFRHsRbEMJj9iHibPSRQMqGTjRsSDQAT7TN1cb1udSf;3pyxch7xmn4w89QCrqWPcb4gC5dciCmidSbNeymq5MfVRDebN1ctFLW1Uqd49us9Nz5MoYigFBq7bkdoTg4qD6bd;3C2ZtwF91WMQpR28e5ujVagePxUrz67NN5raAXBLY6mx8RYL7sitJRXR29S4PRWUP4gt4cqb2eWzax5PgMdmXdGh;X7ywH8mi7KRgpLszNgonoHWWTvUWgQbEp4nVu5AaFJqgrTJM9HtbmkL8VdWqi4QN6F1ogGfKK5QF3D5mZefff1z"
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

	previousSlotID, err := base58.Decode(chunks[4])
	if err != nil {
		return fmt.Errorf("previousSlotID to int: %w", err)
	}

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

	for _, trxIDRaw := range strings.Split(chunks[14], ";") {
		if trxIDRaw == "" || trxIDRaw == "T" {
			continue
		}

		trxID, err := base58.Decode(trxIDRaw)
		if err != nil {
			return fmt.Errorf("transcation id's %q is invalid: %w", trxIDRaw, err)
		}

		b.transactionIDs = append(b.transactionIDs, trxID)
	}

	ctx.activeBank = b
	return nil
}

// BLOCK_END 4 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz 1635424623 1635424624
// BLOCK_END BLOCK_NUM BLOCK_HASH GENESIS_UNIX_TIMESTAMP CLOCK_UNIX_TIMESTAMP
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

	blockHash, err := base58.Decode(chunks[2])
	if err != nil {
		return fmt.Errorf("slot id %q is invalid: %w", chunks[2], err)
	}

	ctx.activeBank.blk.Id = blockHash
	ctx.activeBank.blk.GenesisUnixTimestamp = genesisTimestamp
	ctx.activeBank.blk.ClockUnixTimestamp = clockTimestamp
	//ctx.activeBank.ended = true

	if err := ctx.activeBank.processBatchAggregation(); err != nil {
		return fmt.Errorf("sorting: %w", err)
	}

	ctx.blockBuffer <- ctx.activeBank.blk
	// TODO: it'd be cleaner if this was `nil`, we need to update the tests.
	ctx.activeBank = nil

	zlog.Debug("ctx bank state", zap.Int("bank_count", len(ctx.banks)))

	return nil
}

// BLOCK_ROOT 6482838121
// Simply the root block number, when this block is done processing, and all of its votes are taken into account.
//todo: Block Root not need any more
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
