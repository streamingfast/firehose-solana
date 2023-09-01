package accountsresolver

import (
	"context"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	kvstore "github.com/streamingfast/kvdb/store"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"

	firehose_solana "github.com/streamingfast/firehose-solana"
)

func init() {
	firehose_solana.TestingInitBstream()
}

func Test_ExtendTableLookupInCompiledInstruction(t *testing.T) {
	solBlock := &pbsol.Block{
		PreviousBlockhash: "25anC9GUMtz9KCkAcrdgXX9wG7eS6dnoABNxkMRJx7Ww",
		Blockhash:         "4bDkbonXLmuSXoUazuW455jddkUX4qZEjJR2GNQqapxk",
		ParentSlot:        185_914_861,
		Transactions: []*pbsol.ConfirmedTransaction{
			{
				Transaction: &pbsol.Transaction{
					Signatures: [][]byte{
						{
							218, 119, 81, 113, 241, 238, 10, 186, 4, 63, 230, 112, 8, 234, 164, 13, 182, 68, 243, 66, 13, 60, 168, 233, 122, 194, 100, 216, 3, 141, 252, 236, 168, 29, 11, 167, 162, 148, 43, 83, 24, 137, 71, 46, 167, 201, 222, 83, 82, 203, 192, 227, 116, 68, 241, 151, 97, 28, 129, 21, 36, 191, 193, 0,
						},
					},
					Message: &pbsol.Message{
						Header: nil,
						AccountKeys: [][]byte{
							{
								181, 183, 135, 206, 43, 98, 252, 52, 119, 152, 178, 48, 236, 107, 97, 29, 27, 183, 48, 159, 138, 198, 77, 184, 183, 60, 109, 8, 136, 20, 144, 138,
								232, 6, 118, 13, 89, 67, 215, 220, 13, 136, 192, 84, 125, 140, 127, 32, 29, 89, 3, 51, 236, 126, 230, 134, 162, 189, 230, 203, 71, 233, 183, 226,
								0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
								2, 119, 166, 175, 151, 51, 155, 122, 200, 141, 24, 146, 201, 4, 70, 245, 0, 2, 48, 146, 102, 246, 46, 83, 193, 24, 36, 73, 130, 0, 0, 0,
							},
						},
						RecentBlockhash: nil,
						Instructions: []*pbsol.CompiledInstruction{
							{
								ProgramIdIndex: 0,
								Accounts:       []byte{1, 0, 0, 2},
								Data: []byte{
									2, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 5, 208, 234, 79, 51, 115, 112, 19, 165, 99, 224, 147, 72, 237, 182, 244, 89, 61, 145, 252, 118, 65, 249, 36, 124, 36, 65, 168, 66, 161, 187, 235, 97, 168, 97, 115, 124, 201, 1, 140, 31, 126, 69, 145, 243, 168, 100, 198, 200, 161, 77, 108, 203, 4, 205, 101, 236, 120, 68, 224, 62, 59, 217, 50, 218, 172, 71, 25, 185, 127, 45, 174, 61, 185, 176, 239, 23, 233, 158, 152, 80, 128, 55, 170, 13, 247, 22, 132, 42, 242, 20, 57, 134, 22, 166, 221, 254, 26, 212, 242, 205, 71, 247, 104, 59, 32, 175, 111, 71, 142, 90, 189, 226, 55, 227, 179, 91, 83, 92, 172, 209, 49, 177, 73, 113, 72, 191, 38, 181, 16, 59, 46, 53, 121, 208, 241, 188, 53, 91, 7, 29, 218, 204, 64, 51, 131, 174, 50, 186, 201, 0, 50, 64, 181, 225, 162, 116, 228, 229, 61, 6, 221, 246, 225, 215, 101, 161, 147, 217, 203, 225, 70, 206, 235, 121, 172, 28, 180, 133, 237, 95, 91, 55, 145, 58, 140, 245, 133, 126, 255, 0, 169,
								},
							},
						},
						Versioned:           false,
						AddressTableLookups: nil,
					},
				},
				Meta: &pbsol.TransactionStatusMeta{
					Err:                   nil,
					InnerInstructions:     nil,
					InnerInstructionsNone: false,
				},
			},
		},
		Rewards:   nil,
		BlockTime: nil,
		BlockHeight: &pbsol.BlockHeight{
			BlockHeight: 185_914_862,
		},
		Slot: 185_914_862,
	}
	db, err := kvstore.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)
	cursor := NewCursor(185_914_861, nil)
	p := NewProcessor("test", cursor, NewKVDBAccountsResolver(db), zap.NewNop())
	err = p.ProcessBlock(context.Background(), solBlock)
	require.NoError(t, err)
}

func Test_ExtendTableLookupInInnerInstructionsFromACompiledInstruction(t *testing.T) {

}

func Test_ExtendTableLookupInInnerInstructionsFromAnotherAddressTableLookup(t *testing.T) {

}
