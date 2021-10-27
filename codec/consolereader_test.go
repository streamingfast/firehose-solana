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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_readFromFile(t *testing.T) {
	testPath := "testdata/syncer_20210211"
	cleanup, testdir, err := copyTestDir(testPath, "syncer_20210211")
	require.NoError(t, err)
	defer func() {
		cleanup()
	}()

	cr := testFileConsoleReader(t, fmt.Sprintf("%s/test.dmlog", testPath), testdir)
	s, err := cr.Read()
	require.NoError(t, err)

	block := s.(*pbcodec.Block)

	// TODO: add more testing

	assert.Equal(t, &pbcodec.Block{
		Id:                   "5R9Tn4bNx62TZmfBQzc3MPNyaaANLAuopvnadNrogF1X",
		Number:               41,
		PreviousId:           "AtKUKgTCk5rAxAHEcjSEpm6K5maWDuXwHKz1rvJFFPrK",
		PreviousBlockSlot:    40,
		GenesisUnixTimestamp: 1607616485,
		ClockUnixTimestamp:   1607616485,
		RootNum:              9,
	}, block)
	assert.Equal(t, "5R9Tn4bNx62TZmfBQzc3MPNyaaANLAuopvnadNrogF1X", block.Id)
	assert.Equal(t, uint64(41), block.Number)
	assert.Equal(t, "AtKUKgTCk5rAxAHEcjSEpm6K5maWDuXwHKz1rvJFFPrK", block.PreviousId)
	assert.Equal(t, uint32(1), block.Version)
	assert.Equal(t, uint32(1), block.TransactionCount)
	transaction := block.Transactions[0]
	assert.Equal(t, "2PFKgG8Uq9yHWig6HEEGQUmP8XnpyBi2zeaLAFUKR9rus33QQ1ad1PPmcvGR1hpq77fQEPFmFu8qiMNjmQGbGH6E", transaction.Id)
	assert.Equal(t, 1, len(transaction.Instructions))

}

func Test_processBatchAggregation(t *testing.T) {
	b := &bank{
		transactionIDs: []string{"11", "aa", "cc", "bb", "dd", "ee"},
		blk: &pbcodec.Block{
			Id:     "A",
			Number: 1,
		},
		batchAggregator: [][]*pbcodec.Transaction{
			{
				{Id: "dd"},
			},
			{
				{Id: "ee"},
			},
			{
				{Id: "bb"},
			},
			{
				{Id: "aa"},
				{Id: "cc"},
			},
			{
				{Id: "11"},
			},
		},
	}
	err := b.processBatchAggregation()
	require.NoError(t, err)
	assert.Equal(t, []*pbcodec.Transaction{
		{Id: "11", Index: 0},
		{Id: "aa", Index: 1},
		{Id: "cc", Index: 2},
		{Id: "bb", Index: 3},
		{Id: "dd", Index: 4},
		{Id: "ee", Index: 5},
	}, b.blk.Transactions)
}

func Test_readBlockWork(t *testing.T) {
	tests := []struct {
		name       string
		ctx        *parseCtx
		line       string
		expectCtx  *parseCtx
		expecError bool
	}{
		{
			name: "new full slot work",
			ctx: &parseCtx{
				banks: map[uint64]*bank{},
			},
			line: "BLOCK_WORK 55295939 55295941 full 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 932 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;aa;bb",
			expectCtx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						parentSlotNum:   55295939,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						transactionIDs:  []string{"aa", "bb"},
						blk: &pbcodec.Block{
							Number:            55295941,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
				activeBank: &bank{
					parentSlotNum:   55295939,
					batchAggregator: [][]*pbcodec.Transaction{},
					previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					transactionIDs:  []string{"aa", "bb"},
					blk: &pbcodec.Block{
						Number:            55295941,
						PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						PreviousBlockSlot: 55295939,
					},
				},
			},
		},
		{
			name: "new partial slot work",
			ctx: &parseCtx{
				banks: map[uint64]*bank{},
			},
			line: "BLOCK_WORK 55295939 55295941 partial 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 932 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;aa;bb",
			expectCtx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						parentSlotNum:   55295939,
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						batchAggregator: [][]*pbcodec.Transaction{},
						transactionIDs:  []string{"aa", "bb"},
						blk: &pbcodec.Block{
							Number:            55295941,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
				activeBank: &bank{
					parentSlotNum:   55295939,
					previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					batchAggregator: [][]*pbcodec.Transaction{},
					transactionIDs:  []string{"aa", "bb"},
					blk: &pbcodec.Block{
						Number:            55295941,
						PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						PreviousBlockSlot: 55295939,
					},
				},
			},
		},
		{
			name: "known partial slot work",
			ctx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						parentSlotNum:   55295939,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						transactionIDs:  []string{"aa"},
						blk: &pbcodec.Block{
							Number:            55295941,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
			},
			line: "BLOCK_WORK 55295939 55295941 partial 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 423 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;bb",
			expectCtx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						parentSlotNum:   55295939,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						transactionIDs:  []string{"aa", "bb"},
						blk: &pbcodec.Block{
							Number:            55295941,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
				activeBank: &bank{
					parentSlotNum:   55295939,
					batchAggregator: [][]*pbcodec.Transaction{},
					previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					transactionIDs:  []string{"aa", "bb"},
					blk: &pbcodec.Block{
						Number:            55295941,
						PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						PreviousBlockSlot: 55295939,
					},
				},
			},
		},
		{
			name: "known full slot work",
			ctx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						parentSlotNum:   55295939,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						transactionIDs:  []string{"aa"},
						blk: &pbcodec.Block{
							Number:            55295941,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
			},
			line: "BLOCK_WORK 55295939 55295941 full 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 423 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0 T;bb",
			expectCtx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						parentSlotNum:   55295939,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						transactionIDs:  []string{"aa", "bb"},
						blk: &pbcodec.Block{
							Number:            55295941,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
				activeBank: &bank{
					parentSlotNum:   55295939,
					batchAggregator: [][]*pbcodec.Transaction{},
					previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					transactionIDs:  []string{"aa", "bb"},
					blk: &pbcodec.Block{
						Number:            55295941,
						PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						PreviousBlockSlot: 55295939,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readBlockWork(test.line)
			if test.expecError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCtx, test.ctx)
			}
		})
	}
}

func Test_readBlockEnd(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "end slot",
			ctx: &parseCtx{
				activeBank: &bank{
					transactionIDs: []string{},
					blk: &pbcodec.Block{
						Number:            55295941,
						PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						PreviousBlockSlot: 55295939,
					},
				},
			},
			line: "BLOCK_END 55295941 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz 1606487316 1606487316",
			expectCtx: &parseCtx{
				activeBank: nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readBlockEnd(test.line)
			require.NoError(t, err)
			assert.Equal(t, test.expectCtx, test.ctx)
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
					previousSlotID: "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae",
					ended:          true,
					blk: &pbcodec.Block{
						Id:                   "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
						Number:               55295941,
						Version:              1,
						PreviousId:           "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						PreviousBlockSlot:    55295939,
						GenesisUnixTimestamp: 1606487316,
						ClockUnixTimestamp:   1606487316,
						Transactions: []*pbcodec.Transaction{
							{Id: "a", Index: 0},
							{Id: "b", Index: 1},
							{Id: "c", Index: 2},
							{Id: "d", Index: 3},
						},
						TransactionCount: 4,
					},
				},
				banks: map[uint64]*bank{
					55295941: {
						parentSlotNum:  55295939,
						previousSlotID: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
						ended:          true,
						blk: &pbcodec.Block{
							Id:                   "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
							Number:               55295941,
							Version:              1,
							PreviousId:           "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot:    55295939,
							GenesisUnixTimestamp: 1606487316,
							ClockUnixTimestamp:   1606487316,
							Transactions: []*pbcodec.Transaction{
								{Id: "a", Index: 0},
								{Id: "b", Index: 1},
								{Id: "c", Index: 2},
								{Id: "d", Index: 3},
							},
							TransactionCount: 4,
						},
					},
				},
				blockBuffer: make(chan *pbcodec.Block, 100),
			},
			line: "BANK_ROOT 55295921",
			expectedBlock: &pbcodec.Block{
				Id:         "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
				Number:     55295941,
				PreviousId: "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
				PreviousBlockSlot: 55295939,
				GenesisUnixTimestamp: 1606487316,
				ClockUnixTimestamp:   1606487316,
				Version:    1,
				RootNum: 55295921,
				Transactions: []*pbcodec.Transaction{
					{Id: "a", Index: 0},
					{Id: "b", Index: 1},
					{Id: "c", Index: 2},
					{Id: "d", Index: 3},
				},
				TransactionCount: 4,
			},
			expectCtx: &parseCtx{
				activeBank: nil,
				banks:      map[uint64]*bank{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readBlockRoot(test.line)
			require.NoError(t, err)
			require.Equal(t, 1, len(test.ctx.blockBuffer))
			block := <-test.ctx.blockBuffer
			assert.Equal(t, test.expectedBlock, block)
		})
	}
}

func trxSlice(trxIDs []string) (out []*pbcodec.Transaction) {
	for _, trxID := range trxIDs {
		out = append(out, &pbcodec.Transaction{Id: trxID})
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

	go cr.ProcessData(fl)

	return cr
}

func testReaderConsoleReader(t *testing.T, lines chan string, closer func(), batchFilesPath string) *ConsoleReader {
	t.Helper()

	l := &ConsoleReader{
		lines: lines,
		close: closer,
		ctx:   newParseCtx(batchFilesPath),
	}

	return l
}
