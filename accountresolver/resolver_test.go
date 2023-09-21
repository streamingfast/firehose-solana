package accountsresolver

import (
	"context"
	"os"
	"testing"

	"go.uber.org/zap"

	"github.com/streamingfast/kvdb/store"
	_ "github.com/streamingfast/kvdb/store/badger3"
	"github.com/stretchr/testify/require"
)

var a1 = "2iMPmzAgkUWRjq1E5C4gAFA7bDKCBUrUbogGd8dau5XP"
var a2 = "4YTppbHxaNfZdYjJq9iXvT5T2xnVywqN2FfDX9p7f7MG"
var a3 = "5J7HHVuLb1kUn9q4PZgGYsLm4DNRg1dcmB5FENuM7wQz"
var a4 = "9hT5nqawMAn4xgCcjCmiPDXzVqECQTap3c3wHk6dxyFx"
var a5 = "A8YFwAca6hSp9Xw1RcqUcdXuVgMvQbT2yYLmArCFKxfD"

func TestKVDBAccountsResolver_Extended(t *testing.T) {
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	err = resolver.Extend(context.Background(), 1, []byte{0x00}, accountFromBase58(t, a1), []Account{accountFromBase58(t, a2), accountFromBase58(t, a3)})
	require.NoError(t, err)
	err = resolver.store.FlushPuts(context.Background())
	require.NoError(t, err)

	accounts, _, err := resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 2, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])

	err = resolver.Extend(context.Background(), 100, []byte{0x01}, accountFromBase58(t, a1), []Account{accountFromBase58(t, a4)})
	require.NoError(t, err)
	err = resolver.store.FlushPuts(context.Background())
	require.NoError(t, err)

	accounts, _, err = resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 2, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])

	accounts, _, err = resolver.Resolve(context.Background(), 100, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 3, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])
	require.Equal(t, accountFromBase58(t, a4), accounts[2])

	accounts, _, err = resolver.Resolve(context.Background(), 1000, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 3, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])
	require.Equal(t, accountFromBase58(t, a4), accounts[2])
}

func TestKVDBAccountsResolver_StoreCursor(t *testing.T) {
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	require.NoError(t, err)

	err = resolver.StoreCursor(context.Background(), "r1", NewCursor(1))
	require.NoError(t, err)

	c, err := resolver.GetCursor(context.Background(), "r1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), c.slotNum)
}

func TestKVDBAccountsResolver_StoreCursor_None(t *testing.T) {
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())

	c, err := resolver.GetCursor(context.Background(), "r1")
	require.NoError(t, err)
	require.Nil(t, c)
}

func Test_Extend_Multiple_Accounts_Same_Block(t *testing.T) {
	trxHash1 := []byte{0x01}
	trxHash2 := []byte{0x02}
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	err = resolver.Extend(
		context.Background(),
		1,
		trxHash1,
		accountFromBase58(t, a1),
		[]Account{
			accountFromBase58(t, a2),
			accountFromBase58(t, a3)})

	require.NoError(t, err)
	err = resolver.store.FlushPuts(context.Background())
	require.NoError(t, err)

	accounts, _, err := resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 2, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])

	err = resolver.Extend(
		context.Background(),
		1, trxHash2,
		accountFromBase58(t, a1),
		[]Account{
			accountFromBase58(t, a4),
			accountFromBase58(t, a5),
		})
	require.NoError(t, err)
	err = resolver.store.FlushPuts(context.Background())
	require.NoError(t, err)

	accounts, _, err = resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 4, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])
	require.Equal(t, accountFromBase58(t, a4), accounts[2])
	require.Equal(t, accountFromBase58(t, a5), accounts[3])
}
