package main

import (
	"testing"

	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestAddTransactionConfigs(t *testing.T) {
	source := solanaBlock(t, 100, "block-100",
		transaction("sig-a", nil, nil),
		transaction("sig-b", nil, nil),
	)
	// The source is the truth for everything else, so give the complement a different fee to
	// make sure it is not the one that comes out.
	source.Transactions[0].Meta = &pbsol.TransactionStatusMeta{Fee: 5000}

	complement := solanaBlock(t, 100, "block-100",
		transaction("sig-a", ptr(uint32(1)), &pbsol.TransactionConfig{PriorityFee: ptr(uint64(42))}),
		transaction("sig-b", ptr(uint32(0)), nil),
	)
	complement.Transactions[0].Meta = &pbsol.TransactionStatusMeta{Fee: 9999}

	merged, versionsSet, configsSet, err := addTransactionConfigs(bstreamBlock(t, source), bstreamBlock(t, complement))
	require.NoError(t, err)
	assert.Equal(t, 2, versionsSet)
	assert.Equal(t, 1, configsSet)

	out := solanaPayload(t, merged)

	assert.Equal(t, uint32(1), out.Transactions[0].Transaction.Message.GetVersion())
	assert.Equal(t, uint64(42), out.Transactions[0].Transaction.Message.TransactionConfig.GetPriorityFee())

	assert.Equal(t, uint32(0), out.Transactions[1].Transaction.Message.GetVersion())
	assert.NotNil(t, out.Transactions[1].Transaction.Message.Version)
	assert.Nil(t, out.Transactions[1].Transaction.Message.TransactionConfig)

	assert.Equal(t, uint64(5000), out.Transactions[0].Meta.Fee, "the source is the truth for every other field")
}

func TestAddTransactionConfigsUnsetsWhatTheComplementDoesNotHave(t *testing.T) {
	source := solanaBlock(t, 100, "block-100", transaction("sig-a", ptr(uint32(1)), &pbsol.TransactionConfig{PriorityFee: ptr(uint64(42))}))
	complement := solanaBlock(t, 100, "block-100", transaction("sig-a", nil, nil))

	merged, versionsSet, configsSet, err := addTransactionConfigs(bstreamBlock(t, source), bstreamBlock(t, complement))
	require.NoError(t, err)
	assert.Zero(t, versionsSet)
	assert.Zero(t, configsSet)

	out := solanaPayload(t, merged)
	assert.Nil(t, out.Transactions[0].Transaction.Message.Version)
	assert.Nil(t, out.Transactions[0].Transaction.Message.TransactionConfig)
}

func TestAddTransactionConfigsRejectsMismatchedBlocks(t *testing.T) {
	tests := []struct {
		name       string
		complement *pbsol.Block
		expect     string
	}{
		{
			name:       "transaction count",
			complement: solanaBlock(t, 100, "block-100", transaction("sig-a", nil, nil)),
			expect:     "transaction count mismatch: source has 2, complement has 1",
		},
		{
			name: "transaction identity",
			complement: solanaBlock(t, 100, "block-100",
				transaction("sig-a", nil, nil),
				transaction("sig-other", nil, nil),
			),
			expect: "transaction 1 is not the same in both stores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := solanaBlock(t, 100, "block-100",
				transaction("sig-a", nil, nil),
				transaction("sig-b", nil, nil),
			)

			_, _, _, err := addTransactionConfigs(bstreamBlock(t, source), bstreamBlock(t, tt.complement))
			require.EqualError(t, err, tt.expect)
		})
	}
}

func solanaBlock(t *testing.T, slot uint64, hash string, transactions ...*pbsol.ConfirmedTransaction) *pbsol.Block {
	t.Helper()

	return &pbsol.Block{Slot: slot, Blockhash: hash, Transactions: transactions}
}

func transaction(signature string, version *uint32, config *pbsol.TransactionConfig) *pbsol.ConfirmedTransaction {
	return &pbsol.ConfirmedTransaction{
		Transaction: &pbsol.Transaction{
			Signatures: [][]byte{[]byte(signature)},
			Message: &pbsol.Message{
				Version:           version,
				TransactionConfig: config,
			},
		},
	}
}

func bstreamBlock(t *testing.T, block *pbsol.Block) *pbbstream.Block {
	t.Helper()

	payload, err := anypb.New(block)
	require.NoError(t, err)

	return &pbbstream.Block{Number: block.Slot, Id: block.Blockhash, Payload: payload}
}

func solanaPayload(t *testing.T, block *pbbstream.Block) *pbsol.Block {
	t.Helper()

	out := &pbsol.Block{}
	require.NoError(t, block.Payload.UnmarshalTo(out))

	return out
}

func ptr[T any](in T) *T {
	return &in
}
