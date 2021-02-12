package reader

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/solana-go/programs/serum"
	"github.com/golang/protobuf/proto"

	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/kvdb/store"
	_ "github.com/dfuse-io/kvdb/store/badger"
	"github.com/dfuse-io/solana-go"
	"github.com/stretchr/testify/require"
)

func testKVDBStore(t *testing.T) store.KVStore {
	tmp, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)

	kvStore, err := store.New(fmt.Sprintf("badger://%s/test.db?createTables=true", tmp))
	require.NoError(t, err)
	return kvStore
}

func TestReader_GetOrder(t *testing.T) {

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
			name: "New Order cancelled via close (serum v1)",
		},
		{
			name: "New Order cancelled (serum v2)",
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
			reader := &Reader{testKVDBStore(t)}
			ctx := context.Background()
			for _, kv := range test.data {
				reader.store.Put(ctx, kv.Key, kv.Value)
			}
			reader.store.FlushPuts(ctx)

			out, err := reader.GetInitializeOrder(ctx, test.market, test.orderNum)
			if test.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expect, out)
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
