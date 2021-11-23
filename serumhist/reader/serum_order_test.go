package reader

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	kvdbstore "github.com/streamingfast/kvdb/store"
	_ "github.com/streamingfast/kvdb/store/badger"
	pbserumhist "github.com/streamingfast/sf-solana/pb/sf/solana/serumhist/v1"
	"github.com/streamingfast/sf-solana/serumhist/keyer"
	"github.com/streamingfast/solana-go"
	"github.com/streamingfast/solana-go/programs/serum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testKVDBStore(t *testing.T) (kvdbstore.KVStore, func()) {
	tmp, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)

	kvStore, err := kvdbstore.New(fmt.Sprintf("badger://%s/test.db?createTables=true", tmp))
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
		data        []kvdbstore.KV
		expectError bool
		expect      *pbserumhist.Order
	}{
		{
			name:     "New Order",
			market:   solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"),
			orderNum: 6,
			data: []kvdbstore.KV{
				{
					Key:   keyer.EncodeOrderNew(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 10, 2, 2, 6),
					Value: testNewOrderData(t, 6, solana.MustPublicKeyFromBase58("5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY")),
				},
			},
			expect: &pbserumhist.Order{
				Num:         6,
				Market:      solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW").ToSlice(),
				Trader:      solana.MustPublicKeyFromBase58("5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY").ToSlice(),
				Side:        pbserumhist.Side_ASK,
				LimitPrice:  1955,
				MaxQuantity: 75300000,
				Type:        pbserumhist.OrderType_LIMIT,
				TrxId:       solana.MustPublicKeyFromBase58("4JuADAtnhxg9jUTSx2j7jRQ9vmQiLFTsGxQhQnydHriu1WNbpYhB4LmKn6fmZUL7JTArsSSha8n3zKYpHau4zd5z").ToSlice(),
				TrxIdx:      2,
				InstIdx:     2,
			},
		},
		{
			name:     "New Order cancelled via close (serum v1)",
			market:   solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"),
			orderNum: 6,
			data: []kvdbstore.KV{
				{
					Key:   keyer.EncodeOrderNew(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 10, 2, 2, 6),
					Value: testNewOrderData(t, 6, solana.MustPublicKeyFromBase58("5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY")),
				},
				{
					Key:   keyer.EncodeOrderClose(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 12, 1, 3, 6),
					Value: testInstructionRef(t, "2FmL1EoKvxJjgUkNcMzNpbVMyxeCcFhEsXwNaT3V7eZfGt3d6aTxWkZBt5cr8oqhCyy5SVWmz9YyvuLaWjR4ptnU", timeNow),
				},
			},
			expect: &pbserumhist.Order{
				Num:         6,
				Market:      solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW").ToSlice(),
				Trader:      solana.MustPublicKeyFromBase58("5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY").ToSlice(),
				Side:        pbserumhist.Side_ASK,
				LimitPrice:  1955,
				MaxQuantity: 75300000,
				Type:        pbserumhist.OrderType_LIMIT,
				TrxId:       solana.MustSignatureFromString("4JuADAtnhxg9jUTSx2j7jRQ9vmQiLFTsGxQhQnydHriu1WNbpYhB4LmKn6fmZUL7JTArsSSha8n3zKYpHau4zd5z").ToSlice(),
				TrxIdx:      2,
				InstIdx:     2,
			},
		},
		{
			name:     "New Order cancelled (serum v2)",
			market:   solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"),
			orderNum: 6,
			data: []kvdbstore.KV{
				{
					Key:   keyer.EncodeOrderNew(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 10, 2, 2, 6),
					Value: testNewOrderData(t, 6, solana.MustPublicKeyFromBase58("5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY")),
				},
				{
					Key:   keyer.EncodeOrderCancel(solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW"), 13, 2, 4, 6),
					Value: testInstructionRef(t, "2FmL1EoKvxJjgUkNcMzNpbVMyxeCcFhEsXwNaT3V7eZfGt3d6aTxWkZBt5cr8oqhCyy5SVWmz9YyvuLaWjR4ptnU", timeNow),
				},
			},
			expect: &pbserumhist.Order{
				Num:         6,
				Market:      solana.MustPublicKeyFromBase58("H5uzEytiByuXt964KampmuNCurNDwkVVypkym75J2DQW").ToSlice(),
				Trader:      solana.MustPublicKeyFromBase58("5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY").ToSlice(),
				Side:        pbserumhist.Side_ASK,
				LimitPrice:  1955,
				MaxQuantity: 75300000,
				Type:        pbserumhist.OrderType_LIMIT,
				TrxId:       solana.MustSignatureFromString("4JuADAtnhxg9jUTSx2j7jRQ9vmQiLFTsGxQhQnydHriu1WNbpYhB4LmKn6fmZUL7JTArsSSha8n3zKYpHau4zd5z").ToSlice(),
				TrxIdx:      2,
				InstIdx:     2,
			},
		},
		//{
		//	name: "New Order partial filled",
		//},
		//{
		//	name: "New Order executed via closed (serum v1)",
		//},
		//{
		//	name: "New Order executed (serum v2)",
		//},
		//{
		//	name: "New Order invalid state (fill with canceled)",
		//},
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

			reader := &Reader{store: s}
			order, err := reader.GetOrder(ctx, test.market, test.orderNum)
			if test.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assertProtoEqual(t, test.expect, order)
			}
		})
	}

}

func testNewOrderData(t *testing.T, orderNum uint64, trader solana.PublicKey) []byte {
	o := &pbserumhist.Order{
		Num:         orderNum,
		Market:      nil,
		Trader:      trader.ToSlice(),
		Side:        serum.SideAsk,
		LimitPrice:  1955,
		MaxQuantity: 75300000,
		Type:        pbserumhist.OrderType_LIMIT,
		Fills:       nil,
		TrxId:       solana.MustSignatureFromString("4JuADAtnhxg9jUTSx2j7jRQ9vmQiLFTsGxQhQnydHriu1WNbpYhB4LmKn6fmZUL7JTArsSSha8n3zKYpHau4zd5z").ToSlice(),
	}
	cnt, err := proto.Marshal(o)
	require.NoError(t, err)
	return cnt
}

func testInstructionRef(t *testing.T, trxHash string, timestamp time.Time) []byte {
	o := &pbserumhist.InstructionRef{
		TrxId:     solana.MustSignatureFromString(trxHash).ToSlice(),
		Timestamp: mustProtoTimestamp(timestamp),
	}
	cnt, err := proto.Marshal(o)
	require.NoError(t, err)
	return cnt
}

func mustProtoTimestamp(in time.Time) *timestamp.Timestamp {
	out, err := ptypes.TimestampProto(in)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp conversion %q: %s", in, err))
	}
	return out
}

func assertProtoEqual(t *testing.T, expected, actual proto.Message) {
	t.Helper()

	// We use a custom comparison function and than rely on a standard `assert.Equal` so we get some
	// diffing information. Ideally, a better diff would be displayed, good enough for now.
	if !proto.Equal(expected, actual) {
		assert.Equal(t, expected, actual)
	}
}
