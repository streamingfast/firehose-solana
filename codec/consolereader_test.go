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
	"io"
	"os"
	"strings"
	"testing"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_bank_sortTrx(t *testing.T) {
	b := &bank{
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
	b.sortTrx()
	assert.Equal(t, []*pbcodec.Transaction{
		{Id: "11"},
		{Id: "aa"},
		{Id: "cc"},
		{Id: "bb"},
		{Id: "dd"},
		{Id: "ee"},
	}, b.sortedTrx)
}

func Test_readSlotWork(t *testing.T) {
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
			line: "SLOT_WORK 55295939 55295941 full 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 51936825 932 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0",
			expectCtx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						blockNum:        55295941,
						parentSlotNum:   55295939,
						trxCount:        932,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						sortedTrx:       []*pbcodec.Transaction{},
						slots:           []*pbcodec.Slot{},
						blk: &pbcodec.Block{
							Number:            55295941,
							Height:            51936825,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
				activeBank: &bank{
					blockNum:        55295941,
					parentSlotNum:   55295939,
					trxCount:        932,
					batchAggregator: [][]*pbcodec.Transaction{},
					previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					sortedTrx:       []*pbcodec.Transaction{},
					slots:           []*pbcodec.Slot{},
					blk: &pbcodec.Block{
						Number:            55295941,
						Height:            51936825,
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
			line: "SLOT_WORK 55295939 55295941 partial 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 51936825 932 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0",
			expectCtx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						blockNum:        55295941,
						parentSlotNum:   55295939,
						trxCount:        932,
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						sortedTrx:       []*pbcodec.Transaction{},
						batchAggregator: [][]*pbcodec.Transaction{},
						slots:           []*pbcodec.Slot{},
						blk: &pbcodec.Block{
							Number:            55295941,
							Height:            51936825,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
				activeBank: &bank{
					blockNum:        55295941,
					parentSlotNum:   55295939,
					trxCount:        932,
					previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					sortedTrx:       []*pbcodec.Transaction{},
					batchAggregator: [][]*pbcodec.Transaction{},
					slots:           []*pbcodec.Slot{},
					blk: &pbcodec.Block{
						Number:            55295941,
						Height:            51936825,
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
						blockNum:        55295941,
						parentSlotNum:   55295939,
						trxCount:        932,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						blk: &pbcodec.Block{
							Number:            55295941,
							Height:            51936825,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
			},
			line: "SLOT_WORK 55295939 55295941 partial 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 51936825 423 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0",
			expectCtx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						blockNum:        55295941,
						parentSlotNum:   55295939,
						trxCount:        1355,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						blk: &pbcodec.Block{
							Number:            55295941,
							Height:            51936825,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
				activeBank: &bank{
					blockNum:        55295941,
					parentSlotNum:   55295939,
					trxCount:        1355,
					batchAggregator: [][]*pbcodec.Transaction{},
					previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					blk: &pbcodec.Block{
						Number:            55295941,
						Height:            51936825,
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
						blockNum:        55295941,
						parentSlotNum:   55295939,
						trxCount:        932,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						blk: &pbcodec.Block{
							Number:            55295941,
							Height:            51936825,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
			},
			line: "SLOT_WORK 55295939 55295941 full 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 51936825 423 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0",
			expectCtx: &parseCtx{
				banks: map[uint64]*bank{
					55295941: {
						blockNum:        55295941,
						parentSlotNum:   55295939,
						trxCount:        1355,
						batchAggregator: [][]*pbcodec.Transaction{},
						previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						blk: &pbcodec.Block{
							Number:            55295941,
							Height:            51936825,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
				activeBank: &bank{
					blockNum:        55295941,
					parentSlotNum:   55295939,
					trxCount:        1355,
					batchAggregator: [][]*pbcodec.Transaction{},
					previousSlotID:  "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					blk: &pbcodec.Block{
						Number:            55295941,
						Height:            51936825,
						PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						PreviousBlockSlot: 55295939,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readSlotWork(test.line)
			if test.expecError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCtx, test.ctx)
			}
		})
	}
}

func Test_readSlotEnd(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectSlot  *pbcodec.Slot
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "end slot",
			ctx: &parseCtx{
				activeBank: &bank{
					blockNum:       55295941,
					parentSlotNum:  55295939,
					trxCount:       932,
					previousSlotID: "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae",
					sortedTrx:      trxSlice([]string{"a", "b", "c", "d"}),
					blk: &pbcodec.Block{
						Number:            55295941,
						Height:            51936825,
						PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						PreviousBlockSlot: 55295939,
					},
					slots: []*pbcodec.Slot{
						{
							Id:         "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
							Number:     55295941,
							PreviousId: "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae",
							Version:    1,
							Transactions: []*pbcodec.Transaction{
								{Id: "a", Index: 0, SlotNum: 55295941, SlotHash: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"},
								{Id: "b", Index: 1, SlotNum: 55295941, SlotHash: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"},
								{Id: "c", Index: 2, SlotNum: 55295941, SlotHash: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"},
								{Id: "d", Index: 3, SlotNum: 55295941, SlotHash: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"},
							},
							TransactionCount: 4,
							Block: &pbcodec.Block{
								Id:                   "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
								Number:               55295941,
								Height:               51936825,
								PreviousId:           "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
								PreviousBlockSlot:    55295939,
								GenesisUnixTimestamp: 1606487316,
								ClockUnixTimestamp:   1606487316,
							},
						},
					},
				},
				banks: map[uint64]*bank{
					55295941: {
						blockNum:       55295941,
						parentSlotNum:  55295939,
						trxCount:       932,
						previousSlotID: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
						blk: &pbcodec.Block{
							Number:            55295941,
							Height:            51936825,
							PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
							PreviousBlockSlot: 55295939,
						},
					},
				},
				slotBuffer: make(chan *pbcodec.Slot, 100),
			},
			line: "SLOT_END 55295941 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz 1606487316 1606487316",
			expectSlot: &pbcodec.Slot{
				Id:         "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
				Number:     55295941,
				PreviousId: "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae",
				Version:    1,
				Transactions: []*pbcodec.Transaction{
					{Id: "a", Index: 0, SlotNum: 55295941, SlotHash: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"},
					{Id: "b", Index: 1, SlotNum: 55295941, SlotHash: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"},
					{Id: "c", Index: 2, SlotNum: 55295941, SlotHash: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"},
					{Id: "d", Index: 3, SlotNum: 55295941, SlotHash: "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz"},
				},
				TransactionCount: 4,
				Block: &pbcodec.Block{
					Id:                   "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
					Number:               55295941,
					Height:               51936825,
					PreviousId:           "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					PreviousBlockSlot:    55295939,
					GenesisUnixTimestamp: 1606487316,
					ClockUnixTimestamp:   1606487316,
				},
			},
			expectCtx: &parseCtx{
				activeBank: nil,
				banks:      map[uint64]*bank{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readSlotEnd(test.line)
			require.NoError(t, err)
			assert.Equal(t, 1, len(test.ctx.slotBuffer))
			slot := <-test.ctx.slotBuffer
			assert.Equal(t, test.expectSlot, slot)
		})
	}
}

func Test_readSlotBound(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectSlot  *pbcodec.Slot
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "end slot",
			ctx: &parseCtx{
				activeBank: &bank{
					blockNum:       55295941,
					parentSlotNum:  55295939,
					trxCount:       932,
					previousSlotID: "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
					sortedTrx:      []*pbcodec.Transaction{},
					blk: &pbcodec.Block{
						Number:            55295941,
						Height:            51936825,
						PreviousId:        "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
						PreviousBlockSlot: 55295939,
					},
				},
				slotBuffer: make(chan *pbcodec.Slot, 100),
			},
			line: "SLOT_BOUND 55295940 5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae",
			expectSlot: &pbcodec.Slot{
				Id:               "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae",
				Number:           55295940,
				PreviousId:       "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
				Version:          1,
				TransactionCount: 0,
			},
			expectCtx: &parseCtx{
				activeBank: nil,
				banks:      map[uint64]*bank{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readSlotBound(test.line)
			require.NoError(t, err)
			assert.Equal(t, 1, len(test.ctx.activeBank.slots))
			slot := test.ctx.activeBank.slots[0]
			assert.Equal(t, test.expectSlot, slot)
		})
	}
}

func Test_SimpleSlotWithBound(t *testing.T) {
	expectSlots := []*pbcodec.Slot{
		{
			Id:         "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae",
			Number:     55295940,
			PreviousId: "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
			Version:    1,
			Block: &pbcodec.Block{
				Id:                   "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
				Number:               55295941,
				Height:               51936825,
				PreviousId:           "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
				PreviousBlockSlot:    55295939,
				GenesisUnixTimestamp: 1606487316,
				ClockUnixTimestamp:   1606487316,
			},
		},
		{
			Id:         "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
			Number:     55295941,
			PreviousId: "5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae",
			Version:    1,
			Block: &pbcodec.Block{
				Id:                   "3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz",
				Number:               55295941,
				Height:               51936825,
				PreviousId:           "8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr",
				PreviousBlockSlot:    55295939,
				GenesisUnixTimestamp: 1606487316,
				ClockUnixTimestamp:   1606487316,
			},
		},
	}
	cnt := `
DMLOG SLOT_WORK 55295939 55295941 full 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 51936825 932 814 526 0 0 0 8iCeHcXf6o7Qi8UjYzjoVqo2AUEyo3tpd9V7yVgCesNr 0
DMLOG SLOT_BOUND 55295940 5XcRYrCbLFGSACy43fRdG4zJ88tWxB3eSx36MePjy3Ae
DMLOG SLOT_BOUND 55295941 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz
DMLOG SLOT_END 55295941 3HfUeXfBt8XFHRiyrfhh5EXvFnJTjMHxzemy8DueaUFz 1606487316 1606487316
`

	cr, err := NewConsoleReader(strings.NewReader(cnt))
	require.NoError(t, err)

	for _, expectSlot := range expectSlots {
		o, err := cr.Read()
		if err != io.EOF {
			require.NoError(t, err)
		}
		assert.Equal(t, expectSlot, o)
	}
}

func Test_SimpleSlotFromFile(t *testing.T) {
	t.Skip("till we got new dmlog")
	f, err := os.Open("./test_data/simple.55295915.dmlog")
	require.NoError(t, err)

	cr, err := NewConsoleReader(f)
	require.NoError(t, err)

	s, err := cr.Read()
	require.NoError(t, err)

	slot := s.(*pbcodec.Slot)
	// TODO: add more testing
	assert.Equal(t, "HGRz1p4Eh4wvxWFq8Ki1Jj2uatx2XashQMZhhyMsqNtB", slot.Id)
	assert.Equal(t, "BhGksZQu7eNNRYm9A2ZafCAgGTKwubN4FF68Y2VYq4ET", slot.PreviousId)
	assert.Equal(t, uint64(55295915), slot.Num())
	assert.Equal(t, uint32(465), slot.TransactionCount)
	transaction := slot.Transactions[0]
	assert.Equal(t, "22yEKbnjpxVJQY7RMuvJEYc5PoBVFEFYJCT6Ak2xrtNT7ppzePwneGGuzK2BNEdvdFsvUQHu1qnS688VccHPVKxJ", transaction.Id)
	assert.Equal(t, 1, len(transaction.Instructions))

	_, err = cr.Read()
	assert.Equal(t, err, io.EOF)
}

func Test_VirtualSlotFromFile(t *testing.T) {
	t.Skip("till we got new dmlog")
	f, err := os.Open("./test_data/dual.55295925.dmlog")
	require.NoError(t, err)

	cr, err := NewConsoleReader(f)
	require.NoError(t, err)

	s, err := cr.Read()
	require.NoError(t, err)

	slot := s.(*pbcodec.Slot)
	// TODO: add more testing
	assert.Equal(t, "7DDxS2s6AUJLG66V1SmeQ1zhM8o7vaGAVZvr87TVdYDm", slot.Id)
	assert.Equal(t, "72P3ABBhVV1zR25DxUdFAMmfXf8EMAoSBBZ3tnqJK9nh", slot.PreviousId)
	assert.Equal(t, uint64(55295924), slot.Num())
	assert.Equal(t, uint32(0), slot.TransactionCount)

	s, err = cr.Read()
	require.NoError(t, err)
	slot = s.(*pbcodec.Slot)

	// TODO: add more testing
	assert.Equal(t, "2vyQNcg2ppuEEdV8M9tKouQMNi1iUmgqWTdtLiepyFhJ", slot.Id)
	assert.Equal(t, "7DDxS2s6AUJLG66V1SmeQ1zhM8o7vaGAVZvr87TVdYDm", slot.PreviousId)
	assert.Equal(t, uint64(55295925), slot.Num())
	assert.Equal(t, uint32(538), slot.TransactionCount)
	transaction := slot.Transactions[0]
	assert.Equal(t, "2C2196XJ7seoTfVxa8ToTfPemdW5gzo9i8wyKPEfZ9zL9wxz1PKJRhqZDRNqNQcXKZqZPuGUtDk6MKi8sddZD6Nt", transaction.Id)
	assert.Equal(t, 1, len(transaction.Instructions))

	_, err = cr.Read()
	assert.Equal(t, err, io.EOF)
}

func mustHexDecode(d string) []byte {
	b, e := hex.DecodeString(d)
	if e != nil {
		panic(e)
	}
	return b
}

func trxSlice(trxIDs []string) (out []*pbcodec.Transaction) {
	for _, trxID := range trxIDs {
		out = append(out, &pbcodec.Transaction{Id: trxID})
	}
	return
}
