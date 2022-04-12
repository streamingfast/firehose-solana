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
	"encoding/hex"
	"fmt"
	"github.com/streamingfast/bstream"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/abourget/llerrgroup"
	"github.com/mr-tron/base58"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_processBatchFile(t *testing.T) {
	t.Skip("skip batch file dmlog")
	bank := newBank(10, 9, nil, &options{
		deleteBatchFiles: false,
	})

	bank.processBatchFile("/tmp/solana-test/dmlog-383-1")
	err := bank.errGroup.Wait()
	require.NoError(t, err)
}

func Test_JustRun(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanup, testdir, err := copyTestDir(test.path, "117503050")
			require.NoError(t, err)
			defer func() {
				cleanup()
			}()

			fmt.Println(test.path)
			cr := testFileConsoleReader(t, fmt.Sprintf("%s/test.dmlog", test.path), testdir)
			for {
				_, err = cr.ReadBlock()
				if err == io.EOF {
					return
				}

				require.NoError(t, err)
			}
		})
	}
}

func Test_readFromFile(t *testing.T) {
	t.Skip("test is not present")
	testPath := "testdata/syncer_20210211"
	cleanup, testdir, err := copyTestDir(testPath, "syncer_20210211")
	require.NoError(t, err)
	defer func() {
		cleanup()
	}()

	cr := testFileConsoleReader(t, fmt.Sprintf("%s/test.dmlog", testPath), testdir)
	out, err := cr.ReadBlock()
	require.NoError(t, err)

	blk, err := BlockDecoder(out)
	require.NoError(t, err)
	block := blk.(*pbcodec.Block)

	assert.Equal(t, "JEHPnjb2tV9ELHF8hK8GMU8RgDfWG1dsKmciBrX83RCQ", base58.Encode(block.Id))
	assert.Equal(t, uint64(0), block.Number)
	assert.Equal(t, "11111111111111111111111111111111", base58.Encode(block.PreviousId))
	assert.Equal(t, uint32(1), block.Version)
	assert.Equal(t, uint32(0), block.TransactionCount)

	out, err = cr.ReadBlock()
	require.NoError(t, err)

	blk, err = BlockDecoder(out)
	require.NoError(t, err)
	block = blk.(*pbcodec.Block)

	assert.Equal(t, "EQPhxULMTyjDQWKrbtYm1Mb4zw4VsdK3FLZH8V1uoiLX", base58.Encode(block.Id))
	assert.Equal(t, uint64(1), block.Number)
	assert.Equal(t, "JEHPnjb2tV9ELHF8hK8GMU8RgDfWG1dsKmciBrX83RCQ", base58.Encode(block.PreviousId))
	assert.Equal(t, uint32(1), block.Version)
	assert.Equal(t, uint32(1), block.TransactionCount)
	transaction := block.Transactions[0]
	assert.Equal(t, "4ieqXnsxZuieyUZnyhBfDuwqrYXt48pFaR3M2aoJfARxipSWaVe9FoLBaoLYewF3UvuvMP8bMysvqAPgHiRyBLEP", base58.Encode(transaction.Id))

	for {
		out, err = cr.ReadBlock()
		require.NoError(t, err)

		blk, err = BlockDecoder(out)
		require.NoError(t, err)
		block = blk.(*pbcodec.Block)

		if block.Number == 7 {
			break
		}
	}
}

func Test_processBatchAggregation(t *testing.T) {
	b := &bank{
		transactionIDs: trxIDs(t, "11", "aa", "cc", "bb", "dd", "ee"),
		blk: &pbcodec.Block{
			Id:     blockId(t, "A"),
			Number: 1,
		},
		batchAggregator: [][]*pbcodec.Transaction{
			{
				{Id: trxID(t, "dd")},
			},
			{
				{Id: trxID(t, "ee")},
			},
			{
				{Id: trxID(t, "bb")},
			},
			{
				{Id: trxID(t, "aa")},
				{Id: trxID(t, "cc")},
			},
			{
				{Id: trxID(t, "11")},
			},
		},
	}
	err := b.processBatchAggregation()
	require.NoError(t, err)

	expectOut := []*pbcodec.Transaction{
		{Id: trxID(t, "11"), Index: uint64(0), BeginOrdinal: 0, EndOrdinal: 1},
		{Id: trxID(t, "aa"), Index: uint64(1), BeginOrdinal: 1, EndOrdinal: 2},
		{Id: trxID(t, "cc"), Index: uint64(2), BeginOrdinal: 2, EndOrdinal: 3},
		{Id: trxID(t, "bb"), Index: uint64(3), BeginOrdinal: 3, EndOrdinal: 4},
		{Id: trxID(t, "dd"), Index: uint64(4), BeginOrdinal: 4, EndOrdinal: 5},
		{Id: trxID(t, "ee"), Index: uint64(5), BeginOrdinal: 5, EndOrdinal: 6},
	}
	assert.Equal(t, expectOut, b.blk.Transactions)
}

func MustHexDecode(s string) (out []byte) {
	out, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return
}

func Test_readBlockWork(t *testing.T) {
	parseCtx := &parseCtx{
		banks: map[uint64]*bank{},
	}
	err := parseCtx.readBlockWork("BLOCK_WORK 0 1 full D9i2oNmbRpC3crs3JHw1bWXeRaairC1Ko2QeTYgG2Fte 65 1 64 0 0 0 EnYzNaFkUjkkB475ajS5DanXKTFqnWG8uXNU8nrZ6TyW 0 T;59Hrs5YxFh6amJMQcANFXxoph1oaQYYfwy8tQrBmyihyWwvCyncuXxZEUDS7fEbt2b3BUTB858ucXnLqkTQ2MRPT")
	require.NoError(t, err)
}

//func Test_readBlockWork(t *testing.T) {
//	t.Skip()
//	tests := []struct {
//		name       string
//		ctx        *parseCtx
//		line       string
//		expectCtx  *parseCtx
//		expecError bool
//	}{
//		{
//			name: "new full slot work",
//			ctx: &parseCtx{
//				banks: map[uint64]*bank{},
//			},
//			line: "BLOCK_WORK 55295939 55295941 full 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 932 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;aa;bb",
//			expectCtx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa", "bb"),
//						blk: &pbcodec.Block{
//							Version:       1,
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//				activeBank: &bank{
//					parentSlotNum:   55295939,
//					batchAggregator: [][]*pbcodec.Transaction{},
//					previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//					transactionIDs:  trxIDs(t, "aa", "bb"),
//					blk: &pbcodec.Block{
//						Version:       1,
//						Number:        55295941,
//						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						PreviousBlock: 55295939,
//					},
//				},
//			},
//		},
//		{
//			name: "new partial slot work",
//			ctx: &parseCtx{
//				banks: map[uint64]*bank{},
//			},
//			line: "BLOCK_WORK 55295939 55295941 partial 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 932 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;aa;bb",
//			expectCtx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						batchAggregator: [][]*pbcodec.Transaction{},
//						transactionIDs:  trxIDs(t, "aa", "bb"),
//						blk: &pbcodec.Block{
//							Version:       1,
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//				activeBank: &bank{
//					parentSlotNum:   55295939,
//					previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//					batchAggregator: [][]*pbcodec.Transaction{},
//					transactionIDs:  trxIDs(t, "aa", "bb"),
//					blk: &pbcodec.Block{
//						Version:       1,
//						Number:        55295941,
//						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						PreviousBlock: 55295939,
//					},
//				},
//			},
//		},
//		{
//			name: "known partial slot work",
//			ctx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa"),
//						blk: &pbcodec.Block{
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//			},
//			line: "BLOCK_WORK 55295939 55295941 partial 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 423 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;bb",
//			expectCtx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa", "bb"),
//						blk: &pbcodec.Block{
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//				activeBank: &bank{
//					parentSlotNum:   55295939,
//					batchAggregator: [][]*pbcodec.Transaction{},
//					previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//					transactionIDs:  trxIDs(t, "aa", "bb"),
//					blk: &pbcodec.Block{
//						Number:        55295941,
//						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						PreviousBlock: 55295939,
//					},
//				},
//			},
//		},
//		{
//			name: "known full slot work",
//			ctx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa"),
//						blk: &pbcodec.Block{
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//			},
//			line: "BLOCK_WORK 55295939 55295941 full 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 423 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;bb",
//			expectCtx: &parseCtx{
//				banks: map[uint64]*bank{
//					55295941: {
//						parentSlotNum:   55295939,
//						batchAggregator: [][]*pbcodec.Transaction{},
//						previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						transactionIDs:  trxIDs(t, "aa", "bb"),
//						blk: &pbcodec.Block{
//							Number:        55295941,
//							PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//							PreviousBlock: 55295939,
//						},
//					},
//				},
//				activeBank: &bank{
//					parentSlotNum:   55295939,
//					batchAggregator: [][]*pbcodec.Transaction{},
//					previousSlotID:  blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//					transactionIDs:  trxIDs(t, "aa", "bb"),
//					blk: &pbcodec.Block{
//						Number:        55295941,
//						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
//						PreviousBlock: 55295939,
//					},
//				},
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			err := test.ctx.readBlockWork(test.line)
//			if test.expecError {
//				require.Error(t, err)
//			} else {
//				require.NoError(t, err)
//				assert.Equal(t, test.expectCtx, test.ctx)
//			}
//		})
//	}
//}

func Test_readBlockEnd(t *testing.T) {
	tests := []struct {
		name           string
		ctx            *parseCtx
		line           string
		expectError    bool
		expectBlockID  string
		expectBlockNum uint64
	}{
		{
			name: "end slot",
			ctx: &parseCtx{
				activeBank: &bank{
					transactionIDs: nil,
					blk: &pbcodec.Block{
						Number:        55295941,
						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
						PreviousBlock: 55295939,
					},
					errGroup: llerrgroup.New(10),
				},
				blockBuffer: make(chan *bstream.Block, 1),
				stats:       newParsingStats(55295941),
			},
			line:           "BLOCK_END 55295941 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz 1606487316 1606487316",
			expectBlockID:  "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
			expectBlockNum: 55295941,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readBlockEnd(test.line)
			require.NoError(t, err)
			select {
			case <-time.After(time.Second):
				{
					t.Error("taken too long")
					return
				}
			case block := <-test.ctx.blockBuffer:
				{
					assert.Equal(t, test.expectBlockNum, block.Number)
					assert.Equal(t, test.expectBlockID, block.ID())
				}
			}
			fmt.Println("Done!")
		})
	}
}

func Test_readBlockRoot(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *parseCtx
		line          string
		expectedBlock *pbcodec.Block
		expectCtx     *parseCtx
		expectError   bool
	}{
		{
			name: "block root",
			ctx: &parseCtx{
				activeBank: &bank{
					previousSlotID: blockId(t, "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae"),
					ended:          true,
					blk: &pbcodec.Block{
						Id:                   blockId(t, "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"),
						Number:               55295941,
						Version:              1,
						PreviousId:           blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
						PreviousBlock:        55295939,
						GenesisUnixTimestamp: 1606487316,
						ClockUnixTimestamp:   1606487316,
						Transactions:         trxSlice(t, []string{"aa", "bb", "cc", "dd"}),
						TransactionCount:     4,
					},
				},
				banks: map[uint64]*bank{
					55295941: {
						parentSlotNum:  55295939,
						previousSlotID: blockId(t, "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"),
						ended:          true,
						blk: &pbcodec.Block{
							Id:                   blockId(t, "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"),
							Number:               55295941,
							Version:              1,
							PreviousId:           blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
							PreviousBlock:        55295939,
							GenesisUnixTimestamp: 1606487316,
							ClockUnixTimestamp:   1606487316,
							Transactions:         trxSlice(t, []string{"aa", "bb", "cc", "dd"}),
							TransactionCount:     4,
						},
					},
				},
				blockBuffer: make(chan *bstream.Block, 100),
				stats:       newParsingStats(55295941),
			},
			line: "BANK_ROOT 55295921",
			expectedBlock: &pbcodec.Block{
				Id:                   blockId(t, "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"),
				Number:               55295941,
				PreviousId:           blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
				PreviousBlock:        55295939,
				GenesisUnixTimestamp: 1606487316,
				ClockUnixTimestamp:   1606487316,
				Version:              1,
				Transactions:         trxSlice(t, []string{"aa", "bb", "cc", "dd"}),
				TransactionCount:     4,
			},
			expectCtx: &parseCtx{
				activeBank: nil,
				banks:      map[uint64]*bank{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			go func() {
				select {
				case <-time.After(time.Second):
					{
						t.Errorf("taken too long")
					}
				case block := <-test.ctx.blockBuffer:
					assert.Equal(t, test.expectedBlock, block)

				}
			}()
			err := test.ctx.readBlockRoot(test.line)
			require.NoError(t, err)
			//require.Equal(t, 1, len(test.ctx.blockBuffer))
		})
	}
}

func trxSlice(t *testing.T, trxIDs []string) (out []*pbcodec.Transaction) {
	for i, id := range trxIDs {
		out = append(out, &pbcodec.Transaction{Id: trxID(t, id), Index: uint64(i)})
	}
	return
}

func copyTestDir(testPath, testName string) (func(), string, error) {
	var err error
	var fds []os.FileInfo

	src := fmt.Sprintf("%s/dmlogs", testPath)
	dst, err := ioutil.TempDir("", testName)
	if err != nil {
		return func() {}, "", fmt.Errorf("unable to create test directory: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(dst)
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return cleanup, "", fmt.Errorf("unable to read test data")
	}

	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())
		if !fd.IsDir() {
			if err = copyFile(srcfp, dstfp); err != nil {
				return cleanup, "", fmt.Errorf("unable to copy test file %q to tmp dir %q: %w", srcfp, dstfp, err)
			}
		}
	}
	return cleanup, dst, nil
}

func copyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

func testFileConsoleReader(t *testing.T, dmlogFile, batchFilesPath string) *ConsoleReader {
	t.Helper()

	fl, err := os.Open(dmlogFile)
	require.NoError(t, err)

	cr := testReaderConsoleReader(t, make(chan string, 10000), func() { fl.Close() }, batchFilesPath)

	go processData(cr, fl)

	return cr
}

func testReaderConsoleReader(t *testing.T, lines chan string, closer func(), batchFilesPath string) *ConsoleReader {
	t.Helper()

	l := &ConsoleReader{
		lines: lines,
		close: closer,
		ctx:   newParseCtx(batchFilesPath, nil),
	}

	return l
}

func blockId(t *testing.T, input string) []byte {
	out, err := base58.Decode(input)
	require.NoError(t, err)

	return out
}

func trxIDs(t *testing.T, inputs ...string) [][]byte {
	out := make([][]byte, len(inputs))
	for i, input := range inputs {
		out[i] = trxID(t, input)
	}

	return out
}

func trxID(t *testing.T, input string) []byte {
	bytes, err := hex.DecodeString(input)
	require.NoError(t, err)

	return bytes
}

func processData(r *ConsoleReader, reader io.Reader) error {
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
