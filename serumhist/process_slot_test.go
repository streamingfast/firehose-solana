package serumhist

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	bin "github.com/dfuse-io/binary"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	kvdb "github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/serum"
	"github.com/test-go/testify/assert"
	"google.golang.org/protobuf/proto"
)

func Test_extractOrderSeqNum(t *testing.T) {
	var tests = []struct {
		name         string
		orderID      bin.Uint128
		side         serum.Side
		expectSeqNum uint64
	}{
		{
			name:         "bid, should xor",
			orderID:      bin.Uint128{0xfffffffffffff93b, 1720},
			side:         serum.SideBid,
			expectSeqNum: 1732,
		},
		{
			name:         "ask",
			orderID:      bin.Uint128{1732, 1720},
			side:         serum.SideAsk,
			expectSeqNum: 1732,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectSeqNum, extractOrderSeqNum(test.side, test.orderID))
		})
	}
}

func Test_generateNewOrderKeys(t *testing.T) {
	var tests = []struct {
		name       string
		market     solana.PublicKey
		owner      solana.PublicKey
		slotNumber uint64
		side       serum.Side
		old        *serum.RequestQueue
		new        *serum.RequestQueue
		expect     []*kvdb.KV
	}{
		{
			name:       "request new bid order",
			market:     solana.MustPublicKeyFromBase58("D39ueAqmiu2zT7dHqA2WsH3Vs63dbr98FZa9qDMe6JL8"),
			owner:      solana.MustPublicKeyFromBase58("HHrRXVr6nDDbi3oupMh24bXdzdm6nNAZ9yG3ZA4zBQCV"),
			slotNumber: 2,
			side:       serum.SideBid,
			old: &serum.RequestQueue{
				Head:       0,
				Count:      1,
				NextSeqNum: 7,
				Requests: []*serum.Request{
					{
						RequestFlags: serum.RequestFlagNewOrder,
						OrderID:      bin.Uint128{10, 10},
					},
				},
			},
			new: &serum.RequestQueue{
				Head:       0,
				Count:      2,
				NextSeqNum: 8,
				Requests: []*serum.Request{
					{
						RequestFlags: serum.RequestFlagNewOrder,
						OrderID:      bin.Uint128{10, 10},
					},
					{
						RequestFlags: serum.RequestFlagNewOrder,
						OrderID:      bin.Uint128{0xfffffffffffff93b, 1720},
					},
				},
			},
			expect: []*kvdb.KV{
				{Key: []byte{
					0x03,

					0xf2, 0x0c, 0x2a, 0x97, 0x53, 0x92, 0xa6, 0xd7,
					0x07, 0x93, 0xc3, 0xc4, 0xd6, 0xa0, 0x74, 0xdf,
					0xeb, 0x7e, 0xcb, 0xc4, 0x92, 0x31, 0xa4, 0x9c,
					0xfc, 0xa6, 0xf0, 0xac, 0x38, 0xf2, 0xb0, 0x16,

					0xb2, 0xd9, 0x7a, 0x01, 0xdc, 0x98, 0xf2, 0x65,
					0xb6, 0xd1, 0x1e, 0x87, 0x5b, 0xd9, 0xd4, 0x13,
					0x09, 0x46, 0x18, 0x8e, 0xd6, 0xe9, 0x9d, 0x56,
					0x71, 0xb0, 0x64, 0x33, 0x2a, 0xed, 0xdf, 0x51,

					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xf9, 0x3b,
				}, Value: nil},
				{Key: []byte{
					0x02,

					0xb2, 0xd9, 0x7a, 0x01, 0xdc, 0x98, 0xf2, 0x65,
					0xb6, 0xd1, 0x1e, 0x87, 0x5b, 0xd9, 0xd4, 0x13,
					0x09, 0x46, 0x18, 0x8e, 0xd6, 0xe9, 0x9d, 0x56,
					0x71, 0xb0, 0x64, 0x33, 0x2a, 0xed, 0xdf, 0x51,

					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,

					0xf2, 0x0c, 0x2a, 0x97, 0x53, 0x92, 0xa6, 0xd7,
					0x07, 0x93, 0xc3, 0xc4, 0xd6, 0xa0, 0x74, 0xdf,
					0xeb, 0x7e, 0xcb, 0xc4, 0x92, 0x31, 0xa4, 0x9c,
					0xfc, 0xa6, 0xf0, 0xac, 0x38, 0xf2, 0xb0, 0x16,

					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xf9, 0x3b,
				}, Value: nil},
			},
		},
		{
			name:       "request new ask order",
			market:     solana.MustPublicKeyFromBase58("D39ueAqmiu2zT7dHqA2WsH3Vs63dbr98FZa9qDMe6JL8"),
			owner:      solana.MustPublicKeyFromBase58("HHrRXVr6nDDbi3oupMh24bXdzdm6nNAZ9yG3ZA4zBQCV"),
			slotNumber: 2,
			side:       serum.SideAsk,
			old: &serum.RequestQueue{
				Head:       0,
				Count:      1,
				NextSeqNum: 7,
				Requests: []*serum.Request{
					{
						RequestFlags: serum.RequestFlagNewOrder,
						OrderID:      bin.Uint128{10, 10},
					},
				},
			},
			new: &serum.RequestQueue{
				Head:       0,
				Count:      2,
				NextSeqNum: 8,
				Requests: []*serum.Request{
					{
						RequestFlags: serum.RequestFlagNewOrder,
						OrderID:      bin.Uint128{10, 10},
					},
					{
						RequestFlags: serum.RequestFlagNewOrder,
						OrderID:      bin.Uint128{1732, 1720},
					},
				},
			},
			expect: []*kvdb.KV{
				{Key: []byte{
					0x03,

					0xf2, 0x0c, 0x2a, 0x97, 0x53, 0x92, 0xa6, 0xd7,
					0x07, 0x93, 0xc3, 0xc4, 0xd6, 0xa0, 0x74, 0xdf,
					0xeb, 0x7e, 0xcb, 0xc4, 0x92, 0x31, 0xa4, 0x9c,
					0xfc, 0xa6, 0xf0, 0xac, 0x38, 0xf2, 0xb0, 0x16,

					0xb2, 0xd9, 0x7a, 0x01, 0xdc, 0x98, 0xf2, 0x65,
					0xb6, 0xd1, 0x1e, 0x87, 0x5b, 0xd9, 0xd4, 0x13,
					0x09, 0x46, 0x18, 0x8e, 0xd6, 0xe9, 0x9d, 0x56,
					0x71, 0xb0, 0x64, 0x33, 0x2a, 0xed, 0xdf, 0x51,

					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xf9, 0x3b,
				}, Value: nil},
				{Key: []byte{
					0x02,

					0xb2, 0xd9, 0x7a, 0x01, 0xdc, 0x98, 0xf2, 0x65,
					0xb6, 0xd1, 0x1e, 0x87, 0x5b, 0xd9, 0xd4, 0x13,
					0x09, 0x46, 0x18, 0x8e, 0xd6, 0xe9, 0x9d, 0x56,
					0x71, 0xb0, 0x64, 0x33, 0x2a, 0xed, 0xdf, 0x51,

					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,

					0xf2, 0x0c, 0x2a, 0x97, 0x53, 0x92, 0xa6, 0xd7,
					0x07, 0x93, 0xc3, 0xc4, 0xd6, 0xa0, 0x74, 0xdf,
					0xeb, 0x7e, 0xcb, 0xc4, 0x92, 0x31, 0xa4, 0x9c,
					0xfc, 0xa6, 0xf0, 0xac, 0x38, 0xf2, 0xb0, 0x16,

					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xf9, 0x3b,
				}, Value: nil},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyValues := generateNewOrderKeys(test.slotNumber, test.side, test.market, test.owner, test.old, test.new)
			assert.Equal(t, test.expect, keyValues)
		})
	}
}

func foo() {

}

func Test_generateFillKeys(t *testing.T) {
	var tests = []struct {
		name       string
		market     solana.PublicKey
		slotNumber uint64
		side       serum.Side
		old        *serum.EventQueue
		new        *serum.EventQueue
		expect     []*kvdb.KV
	}{
		//{
		//	name:       "event new bid fill",
		//	market:     solana.MustPublicKeyFromBase58("D39ueAqmiu2zT7dHqA2WsH3Vs63dbr98FZa9qDMe6JL8"),
		//	slotNumber: 2,
		//	side:       serum.SideBid,
		//	old: &serum.EventQueue{
		//		Head:   0,
		//		Count:  1,
		//		SeqNum: 0,
		//		Events: []*serum.Event{
		//			{
		//				Flag:              (serum.EventFlagOut),
		//				OwnerSlot:         1,
		//				FeeTier:           2,
		//				NativeQtyReleased: 3,
		//				NativeQtyPaid:     4,
		//				NativeFeeOrRebate: 5,
		//				OrderID:           bin.Uint128{10, 10},
		//				Owner:             solana.MustPublicKeyFromBase58("G3Di8B5YUeDbSV2hDX9Af5QcfYTXMiy3j5wZCj5AJgoa"),
		//				ClientOrderID:     6,
		//			},
		//		},
		//	},
		//	new: &serum.EventQueue{
		//		Head:   0,
		//		Count:  1,
		//		SeqNum: 0,
		//		Events: []*serum.Event{
		//			{
		//				Flag:              (serum.EventFlagOut),
		//				OwnerSlot:         1,
		//				FeeTier:           2,
		//				NativeQtyReleased: 3,
		//				NativeQtyPaid:     4,
		//				NativeFeeOrRebate: 5,
		//				OrderID:           bin.Uint128{10, 10},
		//				Owner:             solana.MustPublicKeyFromBase58("G3Di8B5YUeDbSV2hDX9Af5QcfYTXMiy3j5wZCj5AJgoa"),
		//				ClientOrderID:     6,
		//			},
		//			{
		//				Flag:              (serum.EventFlagFill | serum.EventFlagBid),
		//				OwnerSlot:         1,
		//				FeeTier:           2,
		//				NativeQtyReleased: 3,
		//				NativeQtyPaid:     4,
		//				NativeFeeOrRebate: 5,
		//				OrderID:           bin.Uint128{0xfffffffffffff93b, 1720},
		//				Owner:             solana.MustPublicKeyFromBase58("HHrRXVr6nDDbi3oupMh24bXdzdm6nNAZ9yG3ZA4zBQCV"),
		//				ClientOrderID:     6,
		//			},
		//		},
		//	},
		//	expect: []*kvdb.KV{
		//		{Key: []byte{
		//			0x01,
		//
		//			0xb2, 0xd9, 0x7a, 0x01, 0xdc, 0x98, 0xf2, 0x65,
		//			0xb6, 0xd1, 0x1e, 0x87, 0x5b, 0xd9, 0xd4, 0x13,
		//			0x09, 0x46, 0x18, 0x8e, 0xd6, 0xe9, 0x9d, 0x56,
		//			0x71, 0xb0, 0x64, 0x33, 0x2a, 0xed, 0xdf, 0x51,
		//
		//			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0xc4,
		//
		//			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
		//		}, Value: mustProto(&pbserumhist.Fill{
		//			Trader:            "HHrRXVr6nDDbi3oupMh24bXdzdm6nNAZ9yG3ZA4zBQCV",
		//			OrderId:           hex.EncodeToString(uint128ToByte(bin.Uint128{Lo: 0xfffffffffffff93b, Hi: 1720})),
		//			Side:              0,
		//			Maker:             false,
		//			NativeQtyPaid:     4,
		//			NativeQtyReceived: 3,
		//			NativeFeeOrRebate: 5,
		//			FeeTier:           2,
		//		})},
		//	},
		//},
		{
			name:       "event new bid - fill & out",
			market:     solana.MustPublicKeyFromBase58("D39ueAqmiu2zT7dHqA2WsH3Vs63dbr98FZa9qDMe6JL8"),
			slotNumber: 2,
			side:       serum.SideBid,
			old: &serum.EventQueue{
				Head:   0,
				Count:  1,
				SeqNum: 0,
				Events: []*serum.Event{},
			},
			new: &serum.EventQueue{
				Head:   0,
				Count:  2,
				SeqNum: 0,
				Events: []*serum.Event{
					{
						Flag:              (serum.EventFlagFill | serum.EventFlagBid | serum.EventFlagMaker),
						OwnerSlot:         1,
						FeeTier:           2,
						NativeQtyReleased: 3,
						NativeQtyPaid:     4,
						NativeFeeOrRebate: 5,
						OrderID:           bin.Uint128{0xfffffffffffff93b, 1720},
						Owner:             solana.MustPublicKeyFromBase58("HHrRXVr6nDDbi3oupMh24bXdzdm6nNAZ9yG3ZA4zBQCV"),
						ClientOrderID:     6,
					},
					{
						Flag:              (serum.EventFlagOut | serum.EventFlagBid),
						OwnerSlot:         1,
						FeeTier:           2,
						NativeQtyReleased: 3,
						NativeQtyPaid:     4,
						NativeFeeOrRebate: 5,
						OrderID:           bin.Uint128{0xfffffffffffff93b, 1720},
						Owner:             solana.MustPublicKeyFromBase58("HHrRXVr6nDDbi3oupMh24bXdzdm6nNAZ9yG3ZA4zBQCV"),
						ClientOrderID:     6,
					},
				},
			},
			expect: []*kvdb.KV{
				{Key: []byte{
					0x01,

					0xb2, 0xd9, 0x7a, 0x01, 0xdc, 0x98, 0xf2, 0x65,
					0xb6, 0xd1, 0x1e, 0x87, 0x5b, 0xd9, 0xd4, 0x13,
					0x09, 0x46, 0x18, 0x8e, 0xd6, 0xe9, 0x9d, 0x56,
					0x71, 0xb0, 0x64, 0x33, 0x2a, 0xed, 0xdf, 0x51,

					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0xc4,

					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
				}, Value: mustProto(&pbserumhist.Fill{
					Trader:            "HHrRXVr6nDDbi3oupMh24bXdzdm6nNAZ9yG3ZA4zBQCV",
					OrderId:           hex.EncodeToString(uint128ToByte(bin.Uint128{Lo: 0xfffffffffffff93b, Hi: 1720})),
					Side:              0,
					Maker:             false,
					NativeQtyPaid:     4,
					NativeQtyReceived: 3,
					NativeFeeOrRebate: 5,
					FeeTier:           2,
				})},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyValues := generateFillKeyValue(test.slotNumber, test.market, test.old, test.new)
			assert.Equal(t, test.expect, keyValues)
		})
	}
}

func pubkeyToSlice(pubkey string) []byte {
	key := solana.MustPublicKeyFromBase58(pubkey)
	return key[:]
}

func uint128ToByte(v bin.Uint128) []byte {
	size := 16
	buf := make([]byte, size)
	binary.LittleEndian.PutUint64(buf, v.Lo)
	binary.LittleEndian.PutUint64(buf[(size/2):], v.Hi)
	return buf
}

func mustProto(message proto.Message) []byte {
	cnt, err := proto.Marshal(message)
	if err != nil {
		panic(err)
	}
	return cnt
}
