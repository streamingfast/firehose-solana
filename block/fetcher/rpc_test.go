package fetcher

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	"github.com/gagliardetto/solana-go"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"

	bin "github.com/streamingfast/binary"
	"github.com/test-go/testify/require"
)

func Test_DoIt(t *testing.T) {
	t.Skip("Only for manual testing")
	//ctx := context.Background()
	//rpcClient := rpc.New(quicknodeURL) //put your own URL in a file call secret.go that will be ignore by git
	//f := NewRPC(rpcClient, 0*time.Millisecond, 0*time.Millisecond, zap.NewNop())
	//_, err := f.Fetch(ctx, 240816742)
	//
	//require.NoError(t, err)
}

func Test_TrxErrorEncode(t *testing.T) {
	cases := []struct {
		name     string
		trxErr   *TransactionError
		expected []byte
	}{
		{
			name: "AccountLoadedTwice",
			trxErr: &TransactionError{
				TrxErrCode: TrxErr_AccountLoadedTwice,
			},
			expected: []byte{1, 0, 0, 0},
		},
		{
			name: "DuplicateInstruction",
			trxErr: &TransactionError{
				TrxErrCode: TrxErr_DuplicateInstruction,
				detail: &DuplicateInstructionError{
					duplicateInstructionIndex: 42,
				},
			},
			expected: []byte{30, 0, 0, 0, 42},
		},
		{
			name: "InsufficientFundsForRent { account_index: u8 }",
			trxErr: &TransactionError{
				TrxErrCode: TrxErr_InsufficientFundsForRent,
				detail: &InsufficientFundsForRentError{
					AccountIndex: 42,
				},
			},
			expected: []byte{31, 0, 0, 0, 42},
		},
		{
			name: "BorshIoError",
			trxErr: &TransactionError{
				TrxErrCode: TrxErr_InstructionError,
				detail: &InstructionError{
					InstructionErrorCode: InstructionError_BorshIoError,
					InstructionIndex:     1,
					detail: &BorshIoError{
						Msg: "error.1",
					},
				},
			},
			expected: []byte{8, 0, 0, 0, 1, 44, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 101, 114, 114, 111, 114, 46, 49},
		},
		{
			name: "custom",
			trxErr: &TransactionError{
				TrxErrCode: TrxErr_InstructionError,
				detail: &InstructionError{
					InstructionErrorCode: 25,
					InstructionIndex:     0,
					detail: InstructionCustomError{
						CustomErrorCode: 42,
					},
				},
			},
			expected: []byte{8, 0, 0, 0, 0, 25, 0, 0, 0, 42, 0, 0, 0},
		},
		{
			name: "custom",
			trxErr: &TransactionError{
				TrxErrCode: TrxErr_InstructionError,
				detail: &InstructionError{
					InstructionErrorCode: 25,
					InstructionIndex:     0,
					detail: InstructionCustomError{
						CustomErrorCode: 0,
					},
				},
			},
			expected: []byte{8, 0, 0, 0, 0, 25, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			encoder := bin.NewEncoder(buf)
			err := c.trxErr.Encode(encoder)
			require.NoError(t, err)
			require.Equal(t, c.expected, buf.Bytes())

		})
	}
}

func Test_InstructionEncode(t *testing.T) {
	cases := []struct {
		name        string
		instruction *InstructionError
		expected    []byte
	}{
		{
			name: "sunny path",
			instruction: &InstructionError{
				InstructionErrorCode: 0,
				InstructionIndex:     1,
				detail:               nil,
			},
			expected: []byte{1, byte(InstructionError_GenericError), 0, 0, 0},
		},
		{
			name: "custom",
			instruction: &InstructionError{
				InstructionErrorCode: 25,
				InstructionIndex:     9,
				detail: InstructionCustomError{
					CustomErrorCode: 6001,
				},
			},
			expected: []byte{9, byte(InstructionError_Custom), 0, 0, 0, 113, 23, 0, 0},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			encoder := bin.NewEncoder(buf)
			err := c.instruction.Encode(encoder)
			require.NoError(t, err)
			require.Equal(t, c.expected, buf.Bytes())

		})
	}
}

func Test_toPbAccountKeys(t *testing.T) {
	accounts := []solana.PublicKey{
		solana.MustPublicKeyFromBase58("EXsJCamTqHJqRqNaB4ZAszGpFw6psMsk9HfjkrrWwJBc"),
		solana.MustPublicKeyFromBase58("8F1yhZvTwrFq5SqJ5PH2VLRRwULUGYHju84FjMtDbJPJ"),
		solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111111"),
	}
	pbAccounts := toPbAccountKeys(accounts)
	expected := [][]byte{
		accounts[0][:],
		accounts[1][:],
		accounts[2][:],
	}
	require.Equal(t, expected, pbAccounts)
}

func Test_toPbMessageVersion(t *testing.T) {
	cases := []struct {
		name     string
		version  solana.MessageVersion
		expected *uint32
	}{
		{
			name:     "legacy message has no wire version",
			version:  solana.MessageVersionLegacy,
			expected: nil,
		},
		{
			name:     "v0 message is wire version 0",
			version:  solana.MessageVersionV0,
			expected: ptr(uint32(0)),
		},
		{
			name:     "v1 message is wire version 1",
			version:  solana.MessageVersionV1,
			expected: ptr(uint32(1)),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expected, toPbMessageVersion(c.version))
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func Test_toPbTransactionConfig(t *testing.T) {
	cases := []struct {
		name     string
		config   solana.TransactionConfig
		expected *pbsol.TransactionConfig
	}{
		{
			name:     "a legacy or v0 message carries no config",
			config:   solana.TransactionConfig{},
			expected: nil,
		},
		{
			name: "a v1 message carrying only a priority fee leaves the rest unset",
			config: solana.TransactionConfig{
				PriorityFee: ptr(uint64(5000)),
			},
			expected: &pbsol.TransactionConfig{
				PriorityFee: ptr(uint64(5000)),
			},
		},
		{
			name: "every config value is carried through",
			config: solana.TransactionConfig{
				PriorityFee:                 ptr(uint64(5000)),
				ComputeUnitLimit:            ptr(uint32(200000)),
				LoadedAccountsDataSizeLimit: ptr(uint32(65536)),
				HeapSize:                    ptr(uint32(262144)),
			},
			expected: &pbsol.TransactionConfig{
				PriorityFee:                 ptr(uint64(5000)),
				ComputeUnitLimit:            ptr(uint32(200000)),
				LoadedAccountsDataSizeLimit: ptr(uint32(65536)),
				HeapSize:                    ptr(uint32(262144)),
			},
		},
		{
			name: "a zero value is kept, since requesting zero differs from not requesting",
			config: solana.TransactionConfig{
				ComputeUnitLimit: ptr(uint32(0)),
			},
			expected: &pbsol.TransactionConfig{
				ComputeUnitLimit: ptr(uint32(0)),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expected, toPbTransactionConfig(c.config))
		})
	}
}

// Test_toPbTransaction_v1_roundtrip walks a v1 transaction all the way from its wire
// bytes to the protobuf the poller writes, which is the path a mainnet block takes.
func Test_toPbTransaction_v1_roundtrip(t *testing.T) {
	payerKey := solana.PrivateKey(ed25519.NewKeyFromSeed(make([]byte, 32)))
	payer := payerKey.PublicKey()
	recipient := solana.MustPublicKeyFromBase58("2mHtsPqiHkQKKh6t2Q1jGwYQ8vG7ULfF7c9k4t9BvGkw")

	transfer := solana.NewInstruction(solana.SystemProgramID, solana.AccountMetaSlice{
		solana.Meta(payer).WRITE().SIGNER(),
		solana.Meta(recipient).WRITE(),
	}, []byte{2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0})

	blockhash := solana.HashFromBytes(bytes.Repeat([]byte{7}, 32))

	tx, err := solana.NewTransaction(
		[]solana.Instruction{transfer},
		blockhash,
		solana.TransactionPayer(payer),
		solana.TransactionV1Config(solana.TransactionConfig{}.
			WithComputeUnitLimit(20_000).
			WithPriorityFee(1_000)),
	)
	require.NoError(t, err)

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payer) {
			return &payerKey
		}
		return nil
	})
	require.NoError(t, err)

	wire, err := tx.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, byte(0x81), wire[0], "a v1 message is prefixed with 0x81")

	decoded, err := solana.TransactionFromBytes(wire)
	require.NoError(t, err)

	out := toPbTransaction(decoded)

	require.True(t, out.Message.Versioned)
	require.Equal(t, ptr(uint32(1)), out.Message.Version)
	require.Empty(t, out.Message.AddressTableLookups, "v1 has no address lookup tables")
	require.Equal(t, blockhash[:], out.Message.RecentBlockhash)
	require.Equal(t, [][]byte{payer.Bytes(), recipient.Bytes(), solana.SystemProgramID.Bytes()}, out.Message.AccountKeys)
	require.Len(t, out.Message.Instructions, 1)
	require.Equal(t, &pbsol.TransactionConfig{
		PriorityFee:      ptr(uint64(1_000)),
		ComputeUnitLimit: ptr(uint32(20_000)),
	}, out.Message.TransactionConfig)
}

// Test_toPbTransaction_legacy_has_no_version pins the difference a consumer relies on to
// tell the formats apart.
func Test_toPbTransaction_legacy_has_no_version(t *testing.T) {
	payer := solana.MustPublicKeyFromBase58("2mHtsPqiHkQKKh6t2Q1jGwYQ8vG7ULfF7c9k4t9BvGkw")
	transfer := solana.NewInstruction(solana.SystemProgramID, solana.AccountMetaSlice{
		solana.Meta(payer).WRITE().SIGNER(),
	}, []byte{2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0})

	tx, err := solana.NewTransaction([]solana.Instruction{transfer}, solana.Hash{}, solana.TransactionPayer(payer))
	require.NoError(t, err)

	out := toPbTransaction(tx)

	require.False(t, out.Message.Versioned)
	require.Nil(t, out.Message.Version)
	require.Nil(t, out.Message.TransactionConfig)
}
