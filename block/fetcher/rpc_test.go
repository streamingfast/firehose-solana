package fetcher

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	bin "github.com/streamingfast/binary"
	"github.com/test-go/testify/require"
	"go.uber.org/zap"
)

func Test_DoIt(t *testing.T) {
	t.Skip("TODO: fix this test")
	ctx := context.Background()
	rpcClient := rpc.New(quicknodeURL) //put your own URL in a file call secret.go that will be ignore by git
	f := NewRPC(rpcClient, 0*time.Millisecond, 0*time.Millisecond, zap.NewNop())
	_, err := f.Fetch(ctx, 240816644)

	require.NoError(t, err)
}

func Test_fetchBlock(t *testing.T) {
	ctx := context.Background()

	f := NewRPC(nil, 0*time.Millisecond, 0*time.Millisecond, zap.NewNop())
	f.fetchBlock = func(ctx context.Context, slot uint64) (uint64, *rpc.GetBlockResult, error) {
		if slot == 240816644 || slot == 240816645 {
			return 0, nil, &jsonrpc.RPCError{
				Code: -32009,
			}
		}
		return slot, &rpc.GetBlockResult{}, nil
	}

	slot, _, err := f.fetch(ctx, 240816644)
	require.NoError(t, err)
	require.Equal(t, uint64(240816646), slot)

}

func Test_fetchBlockFailing(t *testing.T) {
	ctx := context.Background()

	f := NewRPC(nil, 0*time.Millisecond, 0*time.Millisecond, zap.NewNop())
	f.fetchBlock = func(ctx context.Context, slot uint64) (uint64, *rpc.GetBlockResult, error) {
		return 0, nil, &jsonrpc.RPCError{
			Code: -00001,
		}
	}

	slot, b, err := f.fetch(ctx, 240816644)
	var rpcErr *jsonrpc.RPCError
	require.True(t, errors.As(err, &rpcErr))
	require.Nil(t, b)
	require.Equal(t, uint64(0), slot)

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
