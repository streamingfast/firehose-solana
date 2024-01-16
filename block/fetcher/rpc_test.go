package fetcher

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/gagliardetto/solana-go/rpc"
	bin "github.com/streamingfast/binary"
	"github.com/test-go/testify/require"
)

func Test_ToPBTransaction(t *testing.T) {

	b, err := os.ReadFile("/Users/cbillett/devel/sf/firehose-solana/block/fetcher/testdata/result_block_241179689.json")
	require.NoError(t, err)

	getBlockResult := &rpc.GetBlockResult{}
	err = json.Unmarshal(b, getBlockResult)
	require.NoError(t, err)

	_, err = toPbTransactions(getBlockResult.Transactions)
	require.NoError(t, err)

	//trxHash, err := base58.Decode("66gBszm6ybWVVykE4Svm2CvmiSmFbQi2J3Ua2FxHrYL9B1EPsTCGgjfWNVoJHSqd86iKmS8niywSZqDmqkk7uZLM")
	//require.NoError(t, err)
	//for _, tx := range confirmTransactions {
	//	if bytes.Equal(tx.Transaction.Signatures[0], trxHash) {
	//	}
	//}
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
