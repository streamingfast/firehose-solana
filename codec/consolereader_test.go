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
	"os"
	"testing"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_readSlotProcess(t *testing.T) {
	tests := []struct {
		name       string
		ctx        *parseCtx
		line       string
		expectCtx  *parseCtx
		expecError bool
	}{
		{
			name: "process full slot",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{},
			},
			line: "SLOT_PROCESS full 10 bb aa aa 3654731136 53246259 7 10 336 7 57105130 648 601 479",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxMap: map[string]*pbcodec.Transaction{},
						slot: &pbcodec.Slot{
							Id:          "bb",
							Number:      10,
							PreviousId:  "aa",
							Version:     1,
							RootSlotNum: 7,
						},
					},
				},
			},
		},
		{
			name: "process partial slot",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{},
			},
			line: "SLOT_PROCESS partial 10 bb aa aa 3654731136 53246259 7 10 336 7 57105130 648 601 479",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxMap: map[string]*pbcodec.Transaction{},
						slot: &pbcodec.Slot{
							Number:      10,
							PreviousId:  "aa",
							Version:     1,
							RootSlotNum: 7,
						},
					},
				},
			},
		},
		{
			name: "full slot should complete id",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxMap: map[string]*pbcodec.Transaction{},
						slot: &pbcodec.Slot{
							Number:      10,
							PreviousId:  "aa",
							Version:     1,
							RootSlotNum: 7,
						},
					},
				},
			},
			line: "SLOT_PROCESS full 10 cc bb bb 3654731136 53246259 7 10 336 7 57105130 648 601 479",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxMap: map[string]*pbcodec.Transaction{},
						slot: &pbcodec.Slot{
							Id:          "cc",
							Number:      10,
							PreviousId:  "aa",
							Version:     1,
							RootSlotNum: 7,
						},
					},
				},
			},
		},
		{
			name: "process slot num before last ended slot",
			ctx: &parseCtx{
				activeSlots:   map[uint64]*activeSlot{},
				lastEndedSlot: 11,
			},
			line: "SLOT_PROCESS partial 10 bb aa aa 3654731136 53246259 7 10 336 7 57105130 648 601 479",
			expectCtx: &parseCtx{
				activeSlots:   map[uint64]*activeSlot{},
				lastEndedSlot: 11,
			},
		},
		{
			name: "process multiple out of order slots",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					14: {
						trxMap: map[string]*pbcodec.Transaction{},
						slot: &pbcodec.Slot{
							Id:         "ff",
							Number:     14,
							PreviousId: "ee",
							Version:    1,
						},
					},
				},
				lastEndedSlot: 9,
			},
			line: "SLOT_PROCESS full 10 bb aa aa 3654731136 53246259 7 10 336 7 57105130 648 601 479",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					14: {
						trxMap: map[string]*pbcodec.Transaction{},
						slot: &pbcodec.Slot{
							Id:         "ff",
							Number:     14,
							PreviousId: "ee",
							Version:    1,
						},
					},
					10: {
						trxMap: map[string]*pbcodec.Transaction{},
						slot: &pbcodec.Slot{
							RootSlotNum: 7,
							Id:          "bb",
							Number:      10,
							PreviousId:  "aa",
							Version:     1,
						},
					},
				},
				lastEndedSlot: 9,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readSlotProcess(test.line)
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
		name               string
		ctx                *parseCtx
		line               string
		expectSlot         *pbcodec.Slot
		expectLastSlotSeen uint64
		expecError         bool
	}{
		{
			name: "end slot",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						slot: &pbcodec.Slot{
							Id:          "bb",
							Number:      10,
							PreviousId:  "aa",
							Version:     1,
							RootSlotNum: 1,
							Transactions: []*pbcodec.Transaction{
								{Id: "aaa"},
								{Id: "bbb"},
							},
						},
					},
				},
				lastEndedSlot: 9,
			},
			line: "SLOT_END 10 1607211012 1608814366",
			expectSlot: &pbcodec.Slot{
				Id:                   "bb",
				Number:               10,
				PreviousId:           "aa",
				Version:              1,
				RootSlotNum:          1,
				GenesisUnixTimestamp: 1607211012,
				ClockUnixTimestamp:   1608814366,
				Transactions: []*pbcodec.Transaction{
					{Id: "aaa"},
					{Id: "bbb"},
				},
				TransactionCount: 2,
			},
			expectLastSlotSeen: 10,
		},
		{
			name: "end slot that is not active",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{},
			},
			line:       "SLOT_END 10 1607211012 1608814366",
			expecError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			slot, err := test.ctx.readSlotEnd(test.line)
			if test.expecError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectSlot, slot)
			}
		})
	}
}

func Test_readSlotFailed(t *testing.T) {
	tests := []struct {
		name   string
		ctx    *parseCtx
		line   string
		expect error
	}{
		{
			name: "slot failed",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						slot: &pbcodec.Slot{
							Id:          "bb",
							Number:      10,
							PreviousId:  "aa",
							Version:     1,
							RootSlotNum: 1,
							Transactions: []*pbcodec.Transaction{
								{Id: "aaa"},
								{Id: "bbb"},
							},
						},
					},
				},
				lastEndedSlot: 1,
			},
			line:   "SLOT_FAILED 10 unknown",
			expect: fmt.Errorf("slot 10 failed: unknown"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readSlotFailed(test.line)
			assert.Equal(t, test.expect, err)
		})
	}
}

func Test_readTransactionStart(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "golden path",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 0,
						trxMap:   map[string]*pbcodec.Transaction{},
						slot: &pbcodec.Slot{
							Id:     "bb",
							Number: 10,
						},
					},
				},
			},
			line: "TRX_START 10 aaa:bbb:ccc 1 1 2 F8UvVsKnzWyp2nF8aDcqvQ2GVcRpqT91WDsAtvBKCMt9:AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG:SysvarS1otHashes111111111111111111111111111:SysvarC1ock11111111111111111111111111111111:Vote111111111111111111111111111111111111111 dd",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 1,
						slot: &pbcodec.Slot{
							Id:     "bb",
							Number: 10,
						},
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id:                   "aaa",
								SlotNum:              10,
								SlotHash:             "bb",
								AdditionalSignatures: []string{"bbb", "ccc"},
								Header: &pbcodec.MessageHeader{
									NumRequiredSignatures:       1,
									NumReadonlySignedAccounts:   1,
									NumReadonlyUnsignedAccounts: 2,
								},
								AccountKeys: []string{
									"F8UvVsKnzWyp2nF8aDcqvQ2GVcRpqT91WDsAtvBKCMt9",
									"AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG",
									"SysvarS1otHashes111111111111111111111111111",
									"SysvarC1ock11111111111111111111111111111111",
									"Vote111111111111111111111111111111111111111",
								},
								RecentBlockhash: "dd",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readTransactionStart(test.line)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCtx, test.ctx)
			}
		})
	}
}

func Test_readTransactionEnd(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "golden path",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id: "aaa",
							},
						},
						trxIndex: 1,
						slot: &pbcodec.Slot{
							Transactions: []*pbcodec.Transaction{},
						},
					},
				},
			},
			line: "TRX_END 10 aaa",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxMap:   map[string]*pbcodec.Transaction{},
						trxIndex: 1,
						slot: &pbcodec.Slot{
							Transactions: []*pbcodec.Transaction{
								{Id: "aaa"},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readTransactionEnd(test.line)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCtx, test.ctx)
			}
		})
	}
}

func Test_readTransactionLog(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "golden path",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 1,
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id:          "aaa",
								LogMessages: []string{},
							},
						},
					},
				},
			},
			line: "TRX_L 10 aaa aabbcc",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 1,
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id:          "aaa",
								LogMessages: []string{"\xaa\xbb\xcc"},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readTransactionLog(test.line)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCtx, test.ctx)
			}
		})
	}
}

func Test_readInstructionStart(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "golden path",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 1,
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id:           "aaa",
								Instructions: []*pbcodec.Instruction{},
							},
						},
					},
				},
			},
			line: "INST_S 10 aaa 1 0 Vote111111111111111111111111111111111111111 0200000001000000000000000b0000000000000004398c6eecd88cb501e2bd330d15f9810fa76c26f82d165abd0cbb75292ab0e601e64cda5f00000000 Vote111111111111111111111111111111111111111:00;AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG:01;SysvarS1otHashes111111111111111111111111111:00;SysvarC1ock11111111111111111111111111111111:00;F8UvVsKnzWyp2nF8aDcqvQ2GVcRpqT91WDsAtvBKCMt9:11",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 1,
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id: "aaa",
								Instructions: []*pbcodec.Instruction{
									{
										ProgramId: "Vote111111111111111111111111111111111111111",
										AccountKeys: []string{
											"Vote111111111111111111111111111111111111111",
											"AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG",
											"SysvarS1otHashes111111111111111111111111111",
											"SysvarC1ock11111111111111111111111111111111",
											"F8UvVsKnzWyp2nF8aDcqvQ2GVcRpqT91WDsAtvBKCMt9",
										},
										Data:    mustHexDecode("0200000001000000000000000b0000000000000004398c6eecd88cb501e2bd330d15f9810fa76c26f82d165abd0cbb75292ab0e601e64cda5f00000000"),
										Ordinal: 1,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readInstructionStart(test.line)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCtx, test.ctx)
			}
		})
	}
}

func Test_readAccountChange(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "golden path",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 0,
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id: "aaa",
								Instructions: []*pbcodec.Instruction{
									{
										AccountChanges: []*pbcodec.AccountChange{},
									},
								},
							},
						},
					},
				},
			},
			line: "ACCT_CH 10 aaa 1 AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG 01000000d1ee412af80c981c82 012333333333323123123123",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 0,
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id: "aaa",
								Instructions: []*pbcodec.Instruction{
									{
										AccountChanges: []*pbcodec.AccountChange{
											{
												Pubkey:        "AVLN9vwtAtvDFWZJH1jmHi9p2XrRnQKM3bqGy738DKhG",
												PrevData:      mustHexDecode("01000000d1ee412af80c981c82"),
												NewData:       mustHexDecode("012333333333323123123123"),
												NewDataLength: 12,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readAccountChange(test.line)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCtx, test.ctx)
			}
		})
	}
}

func Test_readLamportsChange(t *testing.T) {
	tests := []struct {
		name        string
		ctx         *parseCtx
		line        string
		expectCtx   *parseCtx
		expectError bool
	}{
		{
			name: "golden path",
			ctx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 0,
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id: "aaa",
								Instructions: []*pbcodec.Instruction{
									{
										BalanceChanges: []*pbcodec.BalanceChange{},
									},
								},
							},
						},
					},
				},
			},
			line: "LAMP_CH 10 aaa 1 11111111111111111111111111111111 499999892500 494999892500",
			expectCtx: &parseCtx{
				activeSlots: map[uint64]*activeSlot{
					10: {
						trxIndex: 0,
						trxMap: map[string]*pbcodec.Transaction{
							"aaa": {
								Id: "aaa",
								Instructions: []*pbcodec.Instruction{
									{
										BalanceChanges: []*pbcodec.BalanceChange{
											{
												Pubkey:       "11111111111111111111111111111111",
												PrevLamports: 499999892500,
												NewLamports:  494999892500,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ctx.readLamportsChange(test.line)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectCtx, test.ctx)
			}
		})
	}
}

func Test_fromFile(t *testing.T) {
	t.Skip("Seems this test is not in line with deep mind output")

	f, err := os.Open("./test_data/syncer.dmlog")
	require.NoError(t, err)

	cr, err := NewConsoleReader(f)
	require.NoError(t, err)
	for {
		o, err := cr.Read()
		require.NoError(t, err)
		fmt.Println(o)
	}
}

func mustHexDecode(d string) []byte {
	b, e := hex.DecodeString(d)
	if e != nil {
		panic(e)
	}
	return b
}
