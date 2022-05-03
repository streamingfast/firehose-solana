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
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/streamingfast/sf-solana/types"

	"github.com/abourget/llerrgroup"
	"github.com/golang/protobuf/proto"
	"github.com/mr-tron/base58"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/jsonpb"
	_ "github.com/streamingfast/sf-solana/types"
	pbsolana "github.com/streamingfast/sf-solana/types/pb"
	pbsol "github.com/streamingfast/sf-solana/types/pb/sf/solana/type/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFromFile(t *testing.T) {
	tests := []struct {
		name          string
		deepmindFile  string
		batchFilePath string
		augmented     bool
		expectedErr   error
	}{
		{
			name:         "deepmind standard mode",
			deepmindFile: "testdata/deep-mind-standard.dmlog",
		},
		{
			name:          "deepmind augmented mode",
			deepmindFile:  "testdata/deep-mind-augmented.dmlog",
			batchFilePath: "testdata/deep-mind-augmented-batches",
			augmented:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr := testFileConsoleReader(t, test.deepmindFile, test.batchFilePath)

			buf := &bytes.Buffer{}
			buf.Write([]byte("["))
			first := true

			var reader ObjectReader = func() (interface{}, error) {
				out, err := cr.ReadBlock()
				if err != nil {
					return nil, err
				}

				if test.augmented {
					return out.ToProtocol().(*pbsol.Block), nil
				}
				return out.ToProtocol().(*pbsolana.ConfirmedBlock), nil
			}

			if test.augmented {
				types.SetupSfSolAugmented()
			}
			for {
				out, err := reader()
				if v, ok := out.(proto.Message); ok && !isNil(v) {
					if !first {
						buf.Write([]byte(","))
					}
					first = false

					value, err := jsonpb.MarshalIndentToString(v, "  ")
					require.NoError(t, err)

					buf.Write([]byte(value))
				}

				if err == io.EOF {
					break
				}

				if len(buf.Bytes()) != 0 {
					buf.Write([]byte("\n"))
				}

				if test.expectedErr == nil {
					require.NoError(t, err)
				} else if err != nil {
					require.Equal(t, test.expectedErr, err)
					return
				}
			}
			buf.Write([]byte("]"))

			goldenFile := test.deepmindFile + ".golden.json"
			if os.Getenv("GOLDEN_UPDATE") == "true" {
				ioutil.WriteFile(goldenFile, buf.Bytes(), os.ModePerm)
			}

			cnt, err := ioutil.ReadFile(goldenFile)
			require.NoError(t, err)

			if !assert.JSONEq(t, string(cnt), buf.String()) {
				t.Error("previous diff:\n" + unifiedDiff(t, cnt, buf.Bytes()))
			}
		})
	}
}

func Test_processBatchAggregation(t *testing.T) {
	b := &bank{
		transactionIDs: trxIDs(t, "11", "aa", "cc", "bb", "dd", "ee"),
		blk: &pbsol.Block{
			Id:     blockId(t, "A"),
			Number: 1,
		},
		batchAggregator: [][]*pbsol.Transaction{
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

	expectOut := []*pbsol.Transaction{
		{Id: trxID(t, "11"), Index: uint64(0), BeginOrdinal: 0, EndOrdinal: 1},
		{Id: trxID(t, "aa"), Index: uint64(1), BeginOrdinal: 1, EndOrdinal: 2},
		{Id: trxID(t, "cc"), Index: uint64(2), BeginOrdinal: 2, EndOrdinal: 3},
		{Id: trxID(t, "bb"), Index: uint64(3), BeginOrdinal: 3, EndOrdinal: 4},
		{Id: trxID(t, "dd"), Index: uint64(4), BeginOrdinal: 4, EndOrdinal: 5},
		{Id: trxID(t, "ee"), Index: uint64(5), BeginOrdinal: 5, EndOrdinal: 6},
	}
	assert.Equal(t, expectOut, b.blk.Transactions)
}

func Test_readBlockWork(t *testing.T) {
	parseCtx := &parseCtx{
		logger: zlog,
		banks:  map[uint64]*bank{},
	}
	err := parseCtx.readBlockWork("BLOCK_WORK 0 1 full D9i2oNmbRpC3crs3JHw1bWXeRaairC1Ko2QeTYgG2Fte 65 1 64 0 0 0 EnYzNaFkUjkkB475ajS5DanXKTFqnWG8uXNU8nrZ6TyW 0 T;59Hrs5YxFh6amJMQcANFXxoph1oaQYYfwy8tQrBmyihyWwvCyncuXxZEUDS7fEbt2b3BUTB858ucXnLqkTQ2MRPT")
	require.NoError(t, err)
}

func Test_readInit(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		cr            *ConsoleReader
		expectError   bool
		expectVersion *version
	}{
		{
			name: "golden path",
			line: "INIT 0.1 vanilla-standard 1.9.15 (src:devbuild; feat:1070292356)",
			cr:   &ConsoleReader{logger: zlog},
			expectVersion: &version{
				dmVersion:   "0.1",
				variant:     "vanilla-standard",
				nodeVersion: "1.9.15 (src:devbuild; feat:1070292356)",
			},
		},
		{
			name:        "error invalid log line",
			cr:          &ConsoleReader{logger: zlog},
			line:        "INIT 0 1 vanilla-standard 1.9.15 (src:devbuild; feat:1070292356)",
			expectError: true,
		},
		{
			name:        "duplicate init call",
			line:        "INIT 0.1 vanilla-standard 1.9.15 (src:devbuild; feat:1070292356)",
			cr:          &ConsoleReader{ver: &version{"a", "b", "c"}, logger: zlog},
			expectError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cr.readInit(test.line)
			if test.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectVersion, test.cr.ver)
			}
		})
	}
}

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
				logger: zlog,
				activeBank: &bank{
					transactionIDs: nil,
					blk: &pbsol.Block{
						Number:        55295941,
						PreviousId:    blockId(t, "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr"),
						PreviousBlock: 55295939,
					},
					errGroup: llerrgroup.New(10),
				},
				blockBuffer: make(chan *bstream.Block, 1),
				stats:       newParsingStats(55295941, zlog),
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
		expectedBlock *pbsol.Block
		expectCtx     *parseCtx
		expectError   bool
	}{
		{
			name: "block root",
			ctx: &parseCtx{
				logger: zlog,
				activeBank: &bank{
					previousSlotID: blockId(t, "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae"),
					ended:          true,
					blk: &pbsol.Block{
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
						blk: &pbsol.Block{
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
				stats:       newParsingStats(55295941, zlog),
			},
			line: "BANK_ROOT 55295921",
			expectedBlock: &pbsol.Block{
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

func trxSlice(t *testing.T, trxIDs []string) (out []*pbsol.Transaction) {
	for i, id := range trxIDs {
		out = append(out, &pbsol.Transaction{Id: trxID(t, id), Index: uint64(i)})
	}
	return
}

func testFileConsoleReader(t *testing.T, dmlogFilename, batchFilePath string) *ConsoleReader {
	t.Helper()

	fl, err := os.Open(dmlogFilename)
	require.NoError(t, err)

	cr := testReaderConsoleReader(t, make(chan string, 10000), func() { fl.Close() }, batchFilePath)

	go processData(cr, fl)

	return cr
}

func testReaderConsoleReader(t *testing.T, lines chan string, closer func(), batchFilesPath string) *ConsoleReader {
	t.Helper()
	opts := []ConsoleReaderOption{
		KeepBatchFiles(),
	}
	if batchFilesPath != "" {
		opts = append(opts, WithBatchFilesPath(batchFilesPath))
	}

	cr, err := NewConsoleReader(zlog, lines, opts...)
	require.NoError(t, err)

	cr.close = closer
	return cr
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

func unifiedDiff(t *testing.T, cnt1, cnt2 []byte) string {
	file1 := "/tmp/gotests-linediff-1"
	file2 := "/tmp/gotests-linediff-2"
	err := ioutil.WriteFile(file1, cnt1, 0600)
	require.NoError(t, err)

	err = ioutil.WriteFile(file2, cnt2, 0600)
	require.NoError(t, err)

	cmd := exec.Command("diff", "-u", file1, file2)
	out, _ := cmd.Output()

	return string(out)
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}
