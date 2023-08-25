package solana_accounts_resolver

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/streamingfast/kvdb/store"
	_ "github.com/streamingfast/kvdb/store/badger3"
	"github.com/stretchr/testify/require"
)

var a1 = "2iMPmzAgkUWRjq1E5C4gAFA7bDKCBUrUbogGd8dau5XP"
var a2 = "4YTppbHxaNfZdYjJq9iXvT5T2xnVywqN2FfDX9p7f7MG"
var a3 = "5J7HHVuLb1kUn9q4PZgGYsLm4DNRg1dcmB5FENuM7wQz"
var a4 = "9hT5nqawMAn4xgCcjCmiPDXzVqECQTap3c3wHk6dxyFx"

func testAccountFromBase58(account string) Account {
	data, err := base58.Decode(account)
	if err != nil {
		panic(err)
	}
	return data
}

func TestKVDBAccountsResolver_Extended(t *testing.T) {
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db)
	err = resolver.Extended(context.Background(), 1, testAccountFromBase58(a1), []Account{testAccountFromBase58(a2), testAccountFromBase58(a3)})
	require.NoError(t, err)

	v, err := db.Get(context.Background(), Keys.extendedKeyBytes(testAccountFromBase58(a1), 1))
	require.NoError(t, err)
	fmt.Println("Grrrrr", v)

	accounts, err := resolver.Resolve(context.Background(), 1, testAccountFromBase58(a1))
	require.NoError(t, err)
	require.Equal(t, 2, len(accounts))
}
