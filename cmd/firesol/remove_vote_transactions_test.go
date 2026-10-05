package main

import (
	"testing"

	"github.com/gagliardetto/solana-go"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveVoteTransactions(t *testing.T) {
	failedVote := withAccountKeys(transaction("sig-failed-vote", nil, nil), solana.VoteProgramID)
	failedVote.Meta = &pbsol.TransactionStatusMeta{Err: &pbsol.TransactionError{Err: []byte("failed")}}

	failedTransfer := withAccountKeys(transaction("sig-failed-transfer", nil, nil), solana.SystemProgramID)
	failedTransfer.Meta = &pbsol.TransactionStatusMeta{Err: &pbsol.TransactionError{Err: []byte("failed")}}

	source := solanaBlock(t, 100, "block-100",
		withAccountKeys(transaction("sig-vote", nil, nil), solana.SysVarClockPubkey, solana.VoteProgramID),
		withAccountKeys(transaction("sig-transfer", nil, nil), solana.SystemProgramID),
		failedVote,
		failedTransfer,
		transaction("sig-no-keys", nil, nil),
	)
	source.Rewards = []*pbsol.Reward{{Lamports: 42}}

	out, kept, removed, err := removeVoteTransactions(bstreamBlock(t, source))
	require.NoError(t, err)
	assert.Equal(t, 3, kept)
	assert.Equal(t, 2, removed)

	block := solanaPayload(t, out)

	var signatures []string
	for _, trx := range block.Transactions {
		signatures = append(signatures, string(trx.Transaction.Signatures[0]))
	}
	assert.Equal(t, []string{"sig-transfer", "sig-failed-transfer", "sig-no-keys"}, signatures)
	assert.Equal(t, int64(42), block.Rewards[0].Lamports, "everything but the vote transactions is kept")
}

func withAccountKeys(trx *pbsol.ConfirmedTransaction, keys ...solana.PublicKey) *pbsol.ConfirmedTransaction {
	for _, key := range keys {
		trx.Transaction.Message.AccountKeys = append(trx.Transaction.Message.AccountKeys, key.Bytes())
	}

	return trx
}
