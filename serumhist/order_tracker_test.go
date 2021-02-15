package serumhist

import (
	"context"
	"fmt"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/serum"
	"github.com/golang/protobuf/proto"
	"github.com/test-go/testify/assert"
	"github.com/test-go/testify/require"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func testKVDBStore(t *testing.T) (store.KVStore, func()) {
	tmp, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)

	kvStore, err := store.New(fmt.Sprintf("badger://%s/test.db?createTables=true", tmp))
	require.NoError(t, err)
	return kvStore, func() {
		kvStore.Close()
		err := os.RemoveAll(tmp)
		require.NoError(t, err)
	}
}

func TestReader_GetOrder(t *testing.T) {
	timeNow := time.Now()
	tests := []struct {
		name        string
		market      solana.PublicKey
		orderNum    uint64
		data        []store.KV
		expectError bool
		expect      *pbserumhist.OrderTransition
	}{
		{
			name:     "New Order",
			market:   solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"),
			orderNum: 6,
			data: []store.KV{
				{
					Key:   keyer.EncodeOrderNew(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 10, 2, 2, 6),
					Value: testNewOrderData(t, 6, solana.MustPublicKeyFromBase58("5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY")),
				},
			},
			expect: &pbserumhist.OrderTransition{
				PreviousState: pbserumhist.OrderTransition_STATE_UNKNOWN,
				CurrentState:  pbserumhist.OrderTransition_STATE_APPROVED,
				Transition:    pbserumhist.OrderTransition_TRANS_INIT,
				Order: &pbserumhist.Order{
					Num:         6,
					Market:      "H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW",
					Trader:      "5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY",
					Side:        pbserumhist.Side_ASK,
					LimitPrice:  1955,
					MaxQuantity: 75300000,
					Type:        pbserumhist.OrderType_LIMIT,
					SlotNum:     10,
					SlotHash:    "83Wa21PHcGdzHzVcAiitf4P2D9KjMgNPakTFvnexLuNp",
					TrxId:       "4JuADAtnhxg9jUTSx2j7jRQ9vmQiLFTsGxQhQnydHriu1WNbpYhB4LmKn6fmZUL7JTArsSSha8n3zKYpHau4zd5z",
					TrxIdx:      2,
					InstIdx:     2,
				},
			},
		},
		{
			name:     "New Order cancelled via close (serum v1)",
			market:   solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"),
			orderNum: 6,
			data: []store.KV{
				{
					Key:   keyer.EncodeOrderNew(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 10, 2, 2, 6),
					Value: testNewOrderData(t, 6, solana.MustPublicKeyFromBase58("5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY")),
				},
				{
					Key:   keyer.EncodeOrderClose(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 12, 1, 3, 6),
					Value: testInstructionRef(t, "7gLWGjUvfRnFZa7Z6uxze8JX2mxWC4Td29EiHSN16Ys8", "2FmL1EoKvxJjgUkNcMzNpbVMyxeCcFhEsXwNaT3V7eZfGt3d6aTxWkZBt5cr8oqhCyy5SVWmz9YyvuLaWjR4ptnU", timeNow),
				},
			},
			expect: &pbserumhist.OrderTransition{
				PreviousState: pbserumhist.OrderTransition_STATE_UNKNOWN,
				CurrentState:  pbserumhist.OrderTransition_STATE_CANCELLED,
				Transition:    pbserumhist.OrderTransition_TRANS_INIT,
				Cancellation: &pbserumhist.InstructionRef{
					SlotNum:   12,
					TrxHash:   "2FmL1EoKvxJjgUkNcMzNpbVMyxeCcFhEsXwNaT3V7eZfGt3d6aTxWkZBt5cr8oqhCyy5SVWmz9YyvuLaWjR4ptnU",
					TrxIdx:    1,
					InstIdx:   3,
					SlotHash:  "7gLWGjUvfRnFZa7Z6uxze8JX2mxWC4Td29EiHSN16Ys8",
					Timestamp: mustProtoTimestamp(timeNow),
				},
				Order: &pbserumhist.Order{
					Num:         6,
					Market:      "H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW",
					Trader:      "5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY",
					Side:        pbserumhist.Side_ASK,
					LimitPrice:  1955,
					MaxQuantity: 75300000,
					Type:        pbserumhist.OrderType_LIMIT,
					SlotNum:     10,
					SlotHash:    "83Wa21PHcGdzHzVcAiitf4P2D9KjMgNPakTFvnexLuNp",
					TrxId:       "4JuADAtnhxg9jUTSx2j7jRQ9vmQiLFTsGxQhQnydHriu1WNbpYhB4LmKn6fmZUL7JTArsSSha8n3zKYpHau4zd5z",
					TrxIdx:      2,
					InstIdx:     2,
				},
			},
		},
		{
			name:     "New Order cancelled (serum v2)",
			market:   solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"),
			orderNum: 6,
			data: []store.KV{
				{
					Key:   keyer.EncodeOrderNew(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 10, 2, 2, 6),
					Value: testNewOrderData(t, 6, solana.MustPublicKeyFromBase58("5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY")),
				},
				{
					Key:   keyer.EncodeOrderCancel(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 13, 2, 4, 6),
					Value: testInstructionRef(t, "7gLWGjUvfRnFZa7Z6uxze8JX2mxWC4Td29EiHSN16Ys8", "2FmL1EoKvxJjgUkNcMzNpbVMyxeCcFhEsXwNaT3V7eZfGt3d6aTxWkZBt5cr8oqhCyy5SVWmz9YyvuLaWjR4ptnU", timeNow),
				},
			},
			expect: &pbserumhist.OrderTransition{
				PreviousState: pbserumhist.OrderTransition_STATE_UNKNOWN,
				CurrentState:  pbserumhist.OrderTransition_STATE_CANCELLED,
				Transition:    pbserumhist.OrderTransition_TRANS_INIT,
				Cancellation: &pbserumhist.InstructionRef{
					SlotNum:   13,
					TrxHash:   "2FmL1EoKvxJjgUkNcMzNpbVMyxeCcFhEsXwNaT3V7eZfGt3d6aTxWkZBt5cr8oqhCyy5SVWmz9YyvuLaWjR4ptnU",
					TrxIdx:    2,
					InstIdx:   4,
					SlotHash:  "7gLWGjUvfRnFZa7Z6uxze8JX2mxWC4Td29EiHSN16Ys8",
					Timestamp: mustProtoTimestamp(timeNow),
				},
				Order: &pbserumhist.Order{
					Num:         6,
					Market:      "H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW",
					Trader:      "5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY",
					Side:        pbserumhist.Side_ASK,
					LimitPrice:  1955,
					MaxQuantity: 75300000,
					Type:        pbserumhist.OrderType_LIMIT,
					SlotNum:     10,
					SlotHash:    "83Wa21PHcGdzHzVcAiitf4P2D9KjMgNPakTFvnexLuNp",
					TrxId:       "4JuADAtnhxg9jUTSx2j7jRQ9vmQiLFTsGxQhQnydHriu1WNbpYhB4LmKn6fmZUL7JTArsSSha8n3zKYpHau4zd5z",
					TrxIdx:      2,
					InstIdx:     2,
				},
			},
		},
		{
			name: "New Order partial filled",
		},
		{
			name: "New Order executed via closed (serum v1)",
		},
		{
			name: "New Order executed (serum v2)",
		},
		{
			name: "New Order invalid state (fill with canceled)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, cancel := testKVDBStore(t)
			defer func() {
				cancel()
			}()

			ctx := context.Background()
			for _, kv := range test.data {
				s.Put(ctx, kv.Key, kv.Value)
			}
			s.FlushPuts(ctx)

			_, _, err := GetInitializeOrder(ctx, s, test.market, test.orderNum)
			if test.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				//assert.Equal(t, test.expect, outputTransition)
			}
		})
	}

}

func testNewOrderData(t *testing.T, orderNum uint64, trader solana.PublicKey) []byte {
	o := &pbserumhist.Order{
		Num:         orderNum,
		Market:      "",
		Trader:      trader.String(),
		Side:        serum.SideAsk,
		LimitPrice:  1955,
		MaxQuantity: 75300000,
		Type:        pbserumhist.OrderType_LIMIT,
		Fills:       nil,
		SlotHash:    "83Wa21PHcGdzHzVcAiitf4P2D9KjMgNPakTFvnexLuNp",
		TrxId:       "4JuADAtnhxg9jUTSx2j7jRQ9vmQiLFTsGxQhQnydHriu1WNbpYhB4LmKn6fmZUL7JTArsSSha8n3zKYpHau4zd5z",
	}
	cnt, err := proto.Marshal(o)
	require.NoError(t, err)
	return cnt
}

func testInstructionRef(t *testing.T, slotHash, trxHash string, timestamp time.Time) []byte {
	o := &pbserumhist.InstructionRef{
		SlotHash:  slotHash,
		TrxHash:   trxHash,
		Timestamp: mustProtoTimestamp(timestamp),
	}
	cnt, err := proto.Marshal(o)
	require.NoError(t, err)
	return cnt
}
