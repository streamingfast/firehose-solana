package accountsresolver

import (
	"context"
	"os"
	"testing"

	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	kvstore "github.com/streamingfast/kvdb/store"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func Test_ExtendTableLookupInCompiledInstruction(t *testing.T) {
	tableLookupAccount := accountFromBase58(t, "GcjJQhD7L7esCrjNmkPM8oitsFXRpbWo11LMWfLH89u3")
	tableLookupToExtendIndexFromAccountKeys := byte(1)

	expectedCreatedAccounts := fromBase58Strings(t,
		"PhoeNiXZ8ByJGLkxNfZRnkUfjvmuYqLR89jjFHGqdXY",
		"7aDTsspkQNGKmrexAN7FLx9oxU3iPczSSvHNggyuqYkR",
		"FicF181nDsEcasznMTPp9aLa5Rbpdtd11GtSEa1UUWzx",
		"J6vHZDKghn3dbTG7pcBLzHMnXFoqUEiHVaFfZxojMjXs",
		"DBo9bdufoB8z4FNdNnU8u33SHWNvDa6jFqKcX7NLqTB2",
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
	)

	solBlock := &pbsol.Block{
		PreviousBlockhash: "25anC9GUMtz9KCkAcrdgXX9wG7eS6dnoABNxkMRJx7Ww",
		Blockhash:         "4bDkbonXLmuSXoUazuW455jddkUX4qZEjJR2GNQqapxk",
		ParentSlot:        185_914_861,
		Transactions: []*pbsol.ConfirmedTransaction{
			{
				Transaction: &pbsol.Transaction{
					Signatures: [][]byte{{0}},
					Message: &pbsol.Message{
						AccountKeys: [][]byte{
							accountFromBase58(t, "DEM7JJFjemWE5tjt3aC9eeTsGtTnyAs95EWhY2bM6n1o"),
							tableLookupAccount,
							SystemProgram,
							AddressTableLookupAccountProgram,
						},
						Instructions: []*pbsol.CompiledInstruction{
							{
								ProgramIdIndex: 3,
								Accounts:       []byte{tableLookupToExtendIndexFromAccountKeys, 0, 0, 2},
								Data:           append([]byte{2, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}, encodeAccounts(expectedCreatedAccounts)...),
							},
						},
					},
				},
				Meta: &pbsol.TransactionStatusMeta{
					Err:                   nil,
					InnerInstructions:     nil,
					InnerInstructionsNone: false,
				},
			},
		},
		BlockHeight: &pbsol.BlockHeight{
			BlockHeight: 185_914_862,
		},
		Slot: 185_914_862,
	}
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)
	db, err := kvstore.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	p := NewProcessor("test", NewKVDBAccountsResolver(db, zap.NewNop()), zap.NewNop())
	err = p.ProcessBlock(context.Background(), &Stats{}, solBlock)
	require.NoError(t, err)

	accounts, _, err := resolver.Resolve(context.Background(), 185_914_862, tableLookupAccount)
	require.Equal(t, expectedCreatedAccounts, accounts)
}

func Test_ExtendTableLookup_In_InnerInstructions(t *testing.T) {
	tableLookupAccount := accountFromBase58(t, "6pyNrJXyGdDDA3esoLEHJ2uoohcdf2xGT11acfmfyA7Q")
	tableLookupToExtendIndex := byte(2)

	expectedCreatedAccounts := fromBase58Strings(t,
		"He3iAEV5rYjv6Xf7PxKro19eVrC3QAcdic5CF2D2obPt",
	)
	solBlock := &pbsol.Block{
		PreviousBlockhash: "9RXPunwLvRcNGiLKwMBFtxmqr3d1rTxSkYYsMZPbKCct",
		Blockhash:         "6CqnntW5shmcB92VivDAUkKdckn6m7Dmn7nTzSvX1G6o",
		ParentSlot:        157_564_919,
		Transactions: []*pbsol.ConfirmedTransaction{
			{
				Transaction: &pbsol.Transaction{
					Signatures: [][]byte{{0}},
					Message: &pbsol.Message{
						AccountKeys: [][]byte{
							accountFromBase58(t, "GjtTWjJ6hRemHVP48wMxQ9KrhpayYHLwJtsvWP5G8m2P"),
							accountFromBase58(t, "reGishtXKoJnkn5ZK8WfTmCfmGXSxeGqC6Hat44WYJj"),
							tableLookupAccount,
							accountFromBase58(t, "11111111111111111111111111111111"),
							accountFromBase58(t, "AddressLookupTab1e1111111111111111111111111"),
							accountFromBase58(t, "LTRJikygDHo9aB4Ki2E7phAMSWNwTFJTA5di8nBvRK3"),
						},
						Instructions: []*pbsol.CompiledInstruction{
							{
								ProgramIdIndex: 5,
							},
						},
					},
				},
				Meta: &pbsol.TransactionStatusMeta{
					InnerInstructions: []*pbsol.InnerInstructions{
						{
							Index: 0,
							Instructions: []*pbsol.InnerInstruction{
								{
									ProgramIdIndex: 4,
									Accounts:       []byte{tableLookupToExtendIndex, 15, 0, 3},
									Data:           append([]byte{2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}, encodeAccounts(expectedCreatedAccounts)...),
								},
							},
						},
					},
				},
			},
		},
		BlockHeight: &pbsol.BlockHeight{
			BlockHeight: 157_564_920,
		},
		Slot: 157_564_920,
	}
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)
	db, err := kvstore.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	p := NewProcessor("test", NewKVDBAccountsResolver(db, zap.NewNop()), zap.NewNop())
	err = p.ProcessBlock(context.Background(), &Stats{}, solBlock)
	require.NoError(t, err)

	accounts, _, err := resolver.Resolve(context.Background(), 157_564_921, tableLookupAccount)
	require.Equal(t, expectedCreatedAccounts, accounts)
}

func Test_ExtendTableLookup_By_AnotherAddressTableLookup_Containing_AddressLookupTableProgramID(t *testing.T) {
	tableLookupAddressToExtend := accountFromBase58(t, "GcjJQhD7L7esCrjNmkPM8oitsFXRpbWo11LMWfLH89u3")
	tableLookupAddressToExtendIndex := byte(0)
	tableLookupAddressToResolve := accountFromBase58(t, "6pyNrJXyGdDDA3esoLEHJ2uoohcdf2xGT11acfmfyA7Q")

	expectedCreatedAccounts := fromBase58Strings(t,
		"PhoeNiXZ8ByJGLkxNfZRnkUfjvmuYqLR89jjFHGqdXY",
		"7aDTsspkQNGKmrexAN7FLx9oxU3iPczSSvHNggyuqYkR",
		"FicF181nDsEcasznMTPp9aLa5Rbpdtd11GtSEa1UUWzx",
		"J6vHZDKghn3dbTG7pcBLzHMnXFoqUEiHVaFfZxojMjXs",
		"DBo9bdufoB8z4FNdNnU8u33SHWNvDa6jFqKcX7NLqTB2",
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
	)
	solBlock := &pbsol.Block{
		PreviousBlockhash: "25anC9GUMtz9KCkAcrdgXX9wG7eS6dnoABNxkMRJx7Ww",
		Blockhash:         "4bDkbonXLmuSXoUazuW455jddkUX4qZEjJR2GNQqapxk",
		ParentSlot:        185_914_861,
		Transactions: []*pbsol.ConfirmedTransaction{
			{
				Transaction: &pbsol.Transaction{
					Signatures: [][]byte{{0x01}},
					Message: &pbsol.Message{
						AccountKeys: [][]byte{
							tableLookupAddressToExtend,
							SystemProgram,
						},
						Instructions: []*pbsol.CompiledInstruction{
							{
								ProgramIdIndex: 0,
							},
						},
						AddressTableLookups: []*pbsol.MessageAddressTableLookup{
							{
								AccountKey:      tableLookupAddressToResolve,
								WritableIndexes: []byte{0},
							},
						},
					},
				},
				Meta: &pbsol.TransactionStatusMeta{
					InnerInstructions: []*pbsol.InnerInstructions{
						{
							Index: 0,
							Instructions: []*pbsol.InnerInstruction{
								{
									ProgramIdIndex: 2,
									Accounts:       []byte{tableLookupAddressToExtendIndex, 0, 0, 2},
									Data:           append([]byte{2, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}, encodeAccounts(expectedCreatedAccounts)...),
								},
							},
						},
					},
				},
			},
		},
		BlockHeight: &pbsol.BlockHeight{
			BlockHeight: 185_914_862,
		},
		Slot: 185_914_862,
	}
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)
	db, err := kvstore.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	p := NewProcessor("test", NewKVDBAccountsResolver(db, zap.NewNop()), zap.NewNop())

	err = p.accountsResolver.Extend(context.Background(), 185_914_860, []byte{0x00}, "test1", tableLookupAddressToResolve, Accounts{AddressTableLookupAccountProgram})
	require.NoError(t, err)
	err = resolver.store.FlushPuts(context.Background())
	require.NoError(t, err)

	accounts := NewAccounts(solBlock.Transactions[0].Transaction.Message.AccountKeys)
	require.Equal(t, 2, len(accounts))

	err = p.ProcessBlock(context.Background(), &Stats{}, solBlock)
	require.NoError(t, err)

	accounts = NewAccounts(solBlock.Transactions[0].Transaction.Message.AccountKeys)
	require.Equal(t, 3, len(accounts))
	require.Equal(t, accounts[2], AddressTableLookupAccountProgram)

	accounts, _, err = resolver.Resolve(context.Background(), 185_914_862, tableLookupAddressToExtend)
	require.Equal(t, expectedCreatedAccounts, accounts)

}

func Test_ExtendTableLookup_By_AnotherAddressTableLookup_Containing_ExtendableTableLookup(t *testing.T) {
	tableAccountToExtend := accountFromBase58(t, "GcjJQhD7L7esCrjNmkPM8oitsFXRpbWo11LMWfLH89u3")
	tableLookupToExtendIndex := byte(3)

	tableLookupAccountInTransaction := accountFromBase58(t, "6pyNrJXyGdDDA3esoLEHJ2uoohcdf2xGT11acfmfyA7Q")
	expectedCreatedAccounts := fromBase58Strings(t,
		"PhoeNiXZ8ByJGLkxNfZRnkUfjvmuYqLR89jjFHGqdXY",
		"7aDTsspkQNGKmrexAN7FLx9oxU3iPczSSvHNggyuqYkR",
		"FicF181nDsEcasznMTPp9aLa5Rbpdtd11GtSEa1UUWzx",
		"J6vHZDKghn3dbTG7pcBLzHMnXFoqUEiHVaFfZxojMjXs",
		"DBo9bdufoB8z4FNdNnU8u33SHWNvDa6jFqKcX7NLqTB2",
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
	)
	solBlock := &pbsol.Block{
		PreviousBlockhash: "25anC9GUMtz9KCkAcrdgXX9wG7eS6dnoABNxkMRJx7Ww",
		Blockhash:         "4bDkbonXLmuSXoUazuW455jddkUX4qZEjJR2GNQqapxk",
		ParentSlot:        185_914_861,
		Transactions: []*pbsol.ConfirmedTransaction{
			{
				Transaction: &pbsol.Transaction{
					Signatures: [][]byte{{0x01}},
					Message: &pbsol.Message{
						AccountKeys: [][]byte{
							accountFromBase58(t, "DEM7JJFjemWE5tjt3aC9eeTsGtTnyAs95EWhY2bM6n1o"),
							SystemProgram,
							AddressTableLookupAccountProgram,
						},
						RecentBlockhash: nil,
						Instructions: []*pbsol.CompiledInstruction{
							{
								ProgramIdIndex: 0,
							},
						},
						Versioned: false,
						AddressTableLookups: []*pbsol.MessageAddressTableLookup{
							{
								AccountKey:      tableLookupAccountInTransaction,
								WritableIndexes: []byte{0},
							},
						},
					},
				},
				Meta: &pbsol.TransactionStatusMeta{
					InnerInstructions: []*pbsol.InnerInstructions{
						{
							Index: 0,
							Instructions: []*pbsol.InnerInstruction{
								{
									ProgramIdIndex: 2,
									Accounts:       []byte{tableLookupToExtendIndex, 0, 0, 0},
									Data:           append([]byte{2, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}, encodeAccounts(expectedCreatedAccounts)...),
								},
							},
						},
					},
				},
			},
		},
		BlockHeight: &pbsol.BlockHeight{
			BlockHeight: 185_914_862,
		},
		Slot: 185_914_862,
	}

	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)
	db, err := kvstore.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	p := NewProcessor("test", NewKVDBAccountsResolver(db, zap.NewNop()), zap.NewNop())

	// Pre populate the table lookup account with the address table lookup program
	err = p.accountsResolver.Extend(context.Background(), 185_914_860, []byte{0x00}, "test1", tableLookupAccountInTransaction, Accounts{tableAccountToExtend})
	require.NoError(t, err)
	err = resolver.store.FlushPuts(context.Background())
	require.NoError(t, err)

	err = p.ProcessBlock(context.Background(), &Stats{}, solBlock)
	require.NoError(t, err)

	accounts, _, err := resolver.Resolve(context.Background(), 185_914_862, tableAccountToExtend)
	require.Equal(t, expectedCreatedAccounts, accounts)
}

func Test_ExtendTableLookup_Multiple_Instructions(t *testing.T) {
	tableLookupAccount := accountFromBase58(t, "6XuPWMmiUZXEWaX92eBU1EzVin4JF6pwDWCMtYNAcTre")
	tableLookupToExtendIndex := byte(0)

	firstAccountToAdd := accountFromBase58(t, "EEFdZtowJeqwx83PEFGncJzjJ8QpbedMjxkr18CeFvoZ")
	secondAccountToAdd := accountFromBase58(t, "EEFdZtowJeqwx83PEFGncJzjJ8QpbedMjxkr18CeFvoZ")

	expectedExtendedAccountsForTableLookupAccount := Accounts{firstAccountToAdd, secondAccountToAdd}

	solBlock := &pbsol.Block{
		PreviousBlockhash: "9RXPunwLvRcNGiLKwMBFtxmqr3d1rTxSkYYsMZPbKCct",
		Blockhash:         "6CqnntW5shmcB92VivDAUkKdckn6m7Dmn7nTzSvX1G6o",
		ParentSlot:        157_564_919,
		Transactions: []*pbsol.ConfirmedTransaction{
			{
				Transaction: &pbsol.Transaction{
					Signatures: [][]byte{{0}},
					Message: &pbsol.Message{
						AccountKeys: [][]byte{
							tableLookupAccount,
							accountFromBase58(t, "AddressLookupTab1e1111111111111111111111111"),
						},
						Instructions: []*pbsol.CompiledInstruction{
							{
								ProgramIdIndex: 0,
							},
							{
								ProgramIdIndex: 0,
							},
							{
								ProgramIdIndex: 0,
							},
						},
					},
				},
				Meta: &pbsol.TransactionStatusMeta{
					InnerInstructions: []*pbsol.InnerInstructions{
						{
							Index: 0,
							Instructions: []*pbsol.InnerInstruction{
								{
									ProgramIdIndex: 1,
									Accounts:       []byte{tableLookupToExtendIndex, 15, 0, 3},
									Data:           append([]byte{2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}, encodeAccounts([]Account{firstAccountToAdd})...),
								},
							},
						},
						{
							Index: 2,
							Instructions: []*pbsol.InnerInstruction{
								{
									ProgramIdIndex: 1,
									Accounts:       []byte{tableLookupToExtendIndex, 15, 0, 3},
									Data:           append([]byte{2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}, encodeAccounts([]Account{secondAccountToAdd})...),
								},
							},
						},
					},
				},
			},
		},
		BlockHeight: &pbsol.BlockHeight{
			BlockHeight: 157_564_920,
		},
		Slot: 157_564_920,
	}
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)
	db, err := kvstore.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	p := NewProcessor("test", NewKVDBAccountsResolver(db, zap.NewNop()), zap.NewNop())
	err = p.ProcessBlock(context.Background(), &Stats{}, solBlock)
	require.NoError(t, err)

	accounts, _, err := resolver.Resolve(context.Background(), 157_564_921, tableLookupAccount)
	require.Equal(t, expectedExtendedAccountsForTableLookupAccount, accounts)
}

func Test_BlockResolved(t *testing.T) {
	transactionTableLookupAddress := accountFromBase58(t, "6pyNrJXyGdDDA3esoLEHJ2uoohcdf2xGT11acfmfyA7Q")
	tableContent := fromBase58Strings(t,
		"PhoeNiXZ8ByJGLkxNfZRnkUfjvmuYqLR89jjFHGqdXY",
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
	)
	expectedBlockAccounts := fromBase58Strings(t,
		"DEM7JJFjemWE5tjt3aC9eeTsGtTnyAs95EWhY2bM6n1o",
		"PhoeNiXZ8ByJGLkxNfZRnkUfjvmuYqLR89jjFHGqdXY",
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
	)
	solBlock := &pbsol.Block{
		PreviousBlockhash: "25anC9GUMtz9KCkAcrdgXX9wG7eS6dnoABNxkMRJx7Ww",
		Blockhash:         "4bDkbonXLmuSXoUazuW455jddkUX4qZEjJR2GNQqapxk",
		ParentSlot:        185_914_861,
		Transactions: []*pbsol.ConfirmedTransaction{
			{
				Transaction: &pbsol.Transaction{
					Signatures: [][]byte{{0x01}},
					Message: &pbsol.Message{
						AccountKeys: [][]byte{
							accountFromBase58(t, "DEM7JJFjemWE5tjt3aC9eeTsGtTnyAs95EWhY2bM6n1o"),
						},
						RecentBlockhash: nil,
						Instructions: []*pbsol.CompiledInstruction{
							{
								ProgramIdIndex: 0,
							},
						},
						Versioned: false,
						AddressTableLookups: []*pbsol.MessageAddressTableLookup{
							{
								AccountKey:      transactionTableLookupAddress,
								WritableIndexes: []byte{0},
								ReadonlyIndexes: []byte{1},
							},
						},
					},
				},
				Meta: &pbsol.TransactionStatusMeta{
					InnerInstructions: []*pbsol.InnerInstructions{
						{
							Index: 0,
							Instructions: []*pbsol.InnerInstruction{
								{
									ProgramIdIndex: 0,
									Accounts:       []byte{0},
									Data:           []byte{0},
								},
							},
						},
					},
				},
			},
		},
		BlockHeight: &pbsol.BlockHeight{
			BlockHeight: 185_914_862,
		},
		Slot: 185_914_862,
	}

	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)
	db, err := kvstore.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	p := NewProcessor("test", NewKVDBAccountsResolver(db, zap.NewNop()), zap.NewNop())

	// Pre populate the table lookup account with the address table lookup program
	err = p.accountsResolver.Extend(context.Background(), 185_914_860, []byte{0x00}, "test1", transactionTableLookupAddress, tableContent)
	require.NoError(t, err)
	err = resolver.store.FlushPuts(context.Background())
	require.NoError(t, err)

	err = p.ProcessBlock(context.Background(), &Stats{}, solBlock)
	require.NoError(t, err)

	/*	accounts, _, err := resolver.Resolve(context.Background(), 185_914_862, transactionTableLookupAddress)
		require.Equal(t, tableContent, accounts)
	*/
	blockAccountsData := solBlock.Transactions[0].Transaction.Message.AccountKeys
	blockAccounts := NewAccounts(blockAccountsData)
	require.Equal(t, expectedBlockAccounts, blockAccounts)
}
func Test_BlockResolved_Multiple_extend(t *testing.T) {
	tableLookupAccount := accountFromBase58(t, "6pyNrJXyGdDDA3esoLEHJ2uoohcdf2xGT11acfmfyA7Q")
	tableLookupToExtendIndex := byte(2)

	expectedCreatedAccounts1 := fromBase58Strings(t,
		"He3iAEV5rYjv6Xf7PxKro19eVrC3QAcdic5CF2D2obPt",
	)
	expectedCreatedAccounts2 := fromBase58Strings(t,
		"5Q544fKrFoe6tsEbD7S8EmxGTJYAKtTVhAW5Q5pge4j1",
	)
	solBlock := &pbsol.Block{
		PreviousBlockhash: "9RXPunwLvRcNGiLKwMBFtxmqr3d1rTxSkYYsMZPbKCct",
		Blockhash:         "6CqnntW5shmcB92VivDAUkKdckn6m7Dmn7nTzSvX1G6o",
		ParentSlot:        157_564_919,
		Transactions: []*pbsol.ConfirmedTransaction{
			{
				Transaction: &pbsol.Transaction{
					Signatures: [][]byte{{0}},
					Message: &pbsol.Message{
						AccountKeys: [][]byte{
							accountFromBase58(t, "AddressLookupTab1e1111111111111111111111111"),
						},
						Instructions: []*pbsol.CompiledInstruction{
							{
								ProgramIdIndex: 0,
							},
							{
								ProgramIdIndex: 0,
							},
							{
								ProgramIdIndex: 0,
							},
						},
					},
				},
				Meta: &pbsol.TransactionStatusMeta{
					InnerInstructions: []*pbsol.InnerInstructions{
						{
							Index: 0,
							Instructions: []*pbsol.InnerInstruction{
								{
									ProgramIdIndex: 0,
									Accounts:       []byte{tableLookupToExtendIndex, 15, 0, 3},
									Data:           append([]byte{2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}, encodeAccounts(expectedCreatedAccounts1)...),
								},
							},
						},
						{
							Index: 2,
							Instructions: []*pbsol.InnerInstruction{
								{
									ProgramIdIndex: 0,
									Accounts:       []byte{tableLookupToExtendIndex, 15, 0, 3},
									Data:           append([]byte{2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}, encodeAccounts(expectedCreatedAccounts2)...),
								},
							},
						},
					},
				},
			},
		},
		BlockHeight: &pbsol.BlockHeight{
			BlockHeight: 157_564_920,
		},
		Slot: 157_564_920,
	}
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)
	db, err := kvstore.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	p := NewProcessor("test", NewKVDBAccountsResolver(db, zap.NewNop()), zap.NewNop())
	err = p.ProcessBlock(context.Background(), &Stats{}, solBlock)
	require.NoError(t, err)

	accounts, _, err := resolver.Resolve(context.Background(), 157_564_921, tableLookupAccount)
	require.Equal(t, expectedCreatedAccounts1, accounts)
}
