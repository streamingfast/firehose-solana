package serumhist

import (
	"testing"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go/programs/serum"
	"github.com/test-go/testify/assert"
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
