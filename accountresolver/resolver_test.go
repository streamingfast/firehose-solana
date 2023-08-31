package accountsresolver

import (
	"context"
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

	accounts, _, err := resolver.Resolve(context.Background(), 1, testAccountFromBase58(a1))
	require.NoError(t, err)
	require.Equal(t, 2, len(accounts))
	require.Equal(t, testAccountFromBase58(a2), accounts[0])
	require.Equal(t, testAccountFromBase58(a3), accounts[1])

	err = resolver.Extended(context.Background(), 100, testAccountFromBase58(a1), []Account{testAccountFromBase58(a4)})
	require.NoError(t, err)

	accounts, _, err = resolver.Resolve(context.Background(), 1, testAccountFromBase58(a1))
	require.NoError(t, err)
	require.Equal(t, 2, len(accounts))
	require.Equal(t, testAccountFromBase58(a2), accounts[0])
	require.Equal(t, testAccountFromBase58(a3), accounts[1])

	accounts, _, err = resolver.Resolve(context.Background(), 100, testAccountFromBase58(a1))
	require.NoError(t, err)
	require.Equal(t, 3, len(accounts))
	require.Equal(t, testAccountFromBase58(a2), accounts[0])
	require.Equal(t, testAccountFromBase58(a3), accounts[1])
	require.Equal(t, testAccountFromBase58(a4), accounts[2])

	accounts, _, err = resolver.Resolve(context.Background(), 1000, testAccountFromBase58(a1))
	require.NoError(t, err)
	require.Equal(t, 3, len(accounts))
	require.Equal(t, testAccountFromBase58(a2), accounts[0])
	require.Equal(t, testAccountFromBase58(a3), accounts[1])
	require.Equal(t, testAccountFromBase58(a4), accounts[2])
}

func TestKVDBAccountsResolver_StoreCursor(t *testing.T) {
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db)
	expectedBlockHash, err := base58.Decode("8cv9oNupqL1wKogVHcQpqxC7QPy4SiaRghBiP5U2YYLp")
	require.NoError(t, err)

	err = resolver.StoreCursor(context.Background(), "r1", NewCursor(1, expectedBlockHash))
	require.NoError(t, err)

	c, err := resolver.GetCursor(context.Background(), "r1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), c.slotNum)
	require.Equal(t, expectedBlockHash, c.blockHash)
}

func TestKVDBAccountsResolver_StoreCursor_None(t *testing.T) {
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db)

	c, err := resolver.GetCursor(context.Background(), "r1")
	require.NoError(t, err)
	require.Nil(t, c)
}
