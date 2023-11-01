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
	err = resolver.Extend(
		accountFromBase58(t, a1),
		[]Account{
			accountFromBase58(t, a2),
			accountFromBase58(t, a3),
		},
	)
	require.NoError(t, err)

	err = resolver.CommitBlock(context.Background(), 1)
	require.NoError(t, err)

	// we resolve after the block
	accounts, _, err := resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 0, len(accounts))

	accounts, _, err = resolver.Resolve(context.Background(), 2, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 2, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])

	err = resolver.Extend(accountFromBase58(t, a1), []Account{accountFromBase58(t, a4)})
	require.NoError(t, err)
	err = resolver.CommitBlock(context.Background(), 100)

	accounts, _, err = resolver.Resolve(context.Background(), 100, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 2, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])

	accounts, _, err = resolver.Resolve(context.Background(), 101, accountFromBase58(t, a1))
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
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	err = resolver.Extend(
		accountFromBase58(t, a1),
		[]Account{
			accountFromBase58(t, a2),
			accountFromBase58(t, a3),
		},
	)
	require.NoError(t, err)

	err = resolver.Extend(
		accountFromBase58(t, a1),
		[]Account{
			accountFromBase58(t, a4),
			accountFromBase58(t, a5),
		},
	)
	require.NoError(t, err)

	err = resolver.CommitBlock(context.Background(), 1)
	require.NoError(t, err)

	accounts, _, err := resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 0, len(accounts))

	accounts, _, err = resolver.Resolve(context.Background(), 2, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 4, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])
	require.Equal(t, accountFromBase58(t, a4), accounts[2])
	require.Equal(t, accountFromBase58(t, a5), accounts[3])
}

func Test_Extend_Multiple_Accounts_Same_Trx(t *testing.T) {
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	err = resolver.Extend(
		accountFromBase58(t, a1),
		[]Account{
			accountFromBase58(t, a2),
			accountFromBase58(t, a3)})
	require.NoError(t, err)

	err = resolver.Extend(
		accountFromBase58(t, a1),
		[]Account{
			accountFromBase58(t, a4),
			accountFromBase58(t, a5),
		})
	require.NoError(t, err)

	err = resolver.CommitBlock(context.Background(), 1)
	require.NoError(t, err)

	accounts, _, err := resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 0, len(accounts))

	accounts, _, err = resolver.Resolve(context.Background(), 2, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 4, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
	require.Equal(t, accountFromBase58(t, a3), accounts[1])
	require.Equal(t, accountFromBase58(t, a4), accounts[2])
	require.Equal(t, accountFromBase58(t, a5), accounts[3])
}

func Test_Create_Extend_TableLookupAccount_SameTransaction(t *testing.T) {
	trxHash := []byte{0x01}
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	err = resolver.CreateOrDelete(context.Background(), 1, trxHash, "0", accountFromBase58(t, a1))
	require.NoError(t, err)
	accounts, _, err := resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, Accounts(nil), accounts)

	err = resolver.Extend(accountFromBase58(t, a1), []Account{accountFromBase58(t, a2)})
	require.NoError(t, err)

	err = resolver.CommitBlock(context.Background(), 1)
	require.NoError(t, err)

	accounts, _, err = resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, Accounts(nil), accounts)

	accounts, _, err = resolver.Resolve(context.Background(), 2, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 1, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])
}

func Test_Create_Extend_TableLookupAccount_SameTransaction_Delete_Other_Block(t *testing.T) {
	trxHash, trxHash1 := []byte{0x01}, []byte{0x02}
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	err = resolver.CreateOrDelete(context.Background(), 1, trxHash, "0", accountFromBase58(t, a1))
	require.NoError(t, err)
	accounts, _, err := resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, Accounts(nil), accounts)

	err = resolver.Extend(accountFromBase58(t, a1), []Account{accountFromBase58(t, a2)})
	require.NoError(t, err)

	err = resolver.CommitBlock(context.Background(), 1)

	accounts, _, err = resolver.Resolve(context.Background(), 1, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, Accounts(nil), accounts)

	accounts, _, err = resolver.Resolve(context.Background(), 2, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 1, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])

	err = resolver.CreateOrDelete(context.Background(), 2, trxHash1, "0", accountFromBase58(t, a1))
	require.NoError(t, err)

	accounts, _, err = resolver.Resolve(context.Background(), 2, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, 1, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])

	accounts, _, err = resolver.Resolve(context.Background(), 3, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, Accounts(nil), accounts)
}

func Test_Multiple_Cache_Call(t *testing.T) {
	err := os.RemoveAll("/tmp/my-badger.db")
	require.NoError(t, err)

	db, err := store.New("badger3:///tmp/my-badger.db")
	require.NoError(t, err)

	resolver := NewKVDBAccountsResolver(db, zap.NewNop())
	err = resolver.Extend(
		accountFromBase58(t, a1),
		[]Account{
			accountFromBase58(t, a2),
		})
	require.NoError(t, err)

	err = resolver.CommitBlock(context.Background(), 1)
	require.NoError(t, err)

	//flush the cache
	resolver.cache = make(map[string][]*cacheItem)

	accounts, cached, err := resolver.Resolve(context.Background(), 2, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, false, cached)
	require.Equal(t, 1, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])

	accounts, cached, err = resolver.Resolve(context.Background(), 2, accountFromBase58(t, a1))
	require.NoError(t, err)
	require.Equal(t, true, cached)
	require.Equal(t, 1, len(accounts))
	require.Equal(t, accountFromBase58(t, a2), accounts[0])

}
