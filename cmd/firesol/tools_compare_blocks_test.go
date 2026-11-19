package main

import (
	"testing"

	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func Test_sanitizeSolanaBlockForCompare(t *testing.T) {
	rewards := func(order string, zeroCommission string) *pbbstream.Block {
		a := &pbsol.Reward{Pubkey: "A", Lamports: 0, RewardType: pbsol.RewardType_Voting, Commission: zeroCommission}
		b := &pbsol.Reward{Pubkey: "B", Lamports: 0, RewardType: pbsol.RewardType_Voting}
		c := &pbsol.Reward{Pubkey: "B", Lamports: 5, RewardType: pbsol.RewardType_Staking, Commission: "8"}
		list := []*pbsol.Reward{a, b, c}
		if order == "reversed" {
			list = []*pbsol.Reward{c, b, a}
		}
		payload, err := anypb.New(&pbsol.Block{Slot: 1, Rewards: list})
		require.NoError(t, err)
		return &pbbstream.Block{Number: 1, Payload: payload}
	}
	same := func(x, y *pbbstream.Block) bool {
		return proto.Equal(sanitizeSolanaBlockForCompare(x).Payload, sanitizeSolanaBlockForCompare(y).Payload)
	}

	tests := []struct {
		name           string
		order          bool
		zeroCommission bool
		poller         *pbbstream.Block
		geyser         *pbbstream.Block
		want           bool
	}{
		{"nothing ignored, order differs", false, false, rewards("", ""), rewards("reversed", ""), false},
		{"order ignored", true, false, rewards("", ""), rewards("reversed", ""), true},
		{"order ignored, commission differs", true, false, rewards("", "0"), rewards("reversed", ""), false},
		{"commission ignored, order differs", false, true, rewards("", "0"), rewards("reversed", ""), false},
		{"commission ignored", false, true, rewards("", "0"), rewards("", ""), true},
		{"both ignored", true, true, rewards("", "0"), rewards("reversed", ""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ignoreRewardsOrder, ignoreZeroCommissionFormat = tt.order, tt.zeroCommission
			defer func() { ignoreRewardsOrder, ignoreZeroCommissionFormat = false, false }()

			assert.Equal(t, tt.want, same(tt.poller, tt.geyser))
		})
	}
}
