package serumhist

import (
	"testing"

	kvdb "github.com/dfuse-io/kvdb/store"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go/programs/serum"
)

func Test_ProcessRequestQueue(t *testing.T) {
	tests := []struct {
		name   string
		old    *serum.RequestQueue
		new    *serum.RequestQueue
		expect []kvdb.KV
	}{
		{
			name: "request new order added",
			old: &serum.RequestQueue{
				Head:       0,
				Count:      1,
				NextSeqNum: 7,
				Requests: []*serum.Request{
					{
						RequestFlags: serum.RequestFlagNewOrder,
						OrderId:      bin.Uint128{10, 10},
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
						OrderId:      bin.Uint128{10, 10},
					},
					{
						RequestFlags: serum.RequestFlagNewOrder,
						OrderId:      bin.Uint128{20, 20},
					},
				},
			},
			expect: []kvdb.KV{},
		},
		//{
		//	name: "request new order removed",
		//},
		//{
		//	name: "request cancel order added",
		//},
		//{
		//	name: "request cancel order canceled",
		//},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// keyValue := getRequestQueueChangeKeys(test.old, test.new)
			// fmt.Println(keyValue)
		})
	}
}
