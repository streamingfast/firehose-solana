package solana_accounts_resolver

import (
	"context"
	"fmt"

	"github.com/streamingfast/kvdb/store"
)

type AccountsResolver interface {
	Extended(ctx context.Context, blockNum uint64, key Account, accounts Accounts) error
	Resolve(ctx context.Context, blockNum uint64, key Account) (Accounts, error)
}

type KVDBAccountsResolver struct {
	store store.KVStore
}

func NewKVDBAccountsResolver(store store.KVStore) *KVDBAccountsResolver {
	return &KVDBAccountsResolver{
		store: store,
	}
}

func (k *KVDBAccountsResolver) Extended(ctx context.Context, blockNum uint64, key Account, accounts Accounts) error {
	currentAccounts, err := k.Resolve(ctx, blockNum, key)
	if err != nil {
		return fmt.Errorf("retreiving last accounts for key %q: %w", key, err)
	}

	payload := encodeAccounts(append(currentAccounts, accounts...))
	err = k.store.Put(ctx, Keys.extendedKeyBytes(key, blockNum), payload)
	if err != nil {
		return fmt.Errorf("writing extended accounts for key %q: %w", key, err)
	}
	err = k.store.FlushPuts(ctx)
	if err != nil {
		return fmt.Errorf("flushing extended accounts for key %q: %w", key, err)
	}

	return nil
}

func (k *KVDBAccountsResolver) Resolve(ctx context.Context, atBlockNum uint64, key Account) (Accounts, error) {
	//	iter := db.Prefix(context.Background(), Keys.lookupPrefixBytes(testAccountFromBase58(a1)), store.Unlimited)
	//	require.NoError(t, iter.Err())
	//	for iter.Next() {
	//		fmt.Println("fuck!", iter.Item().Key)
	//		fmt.Println("fuck value", iter.Item().Value)
	//	}
	keyBytes := Keys.lookupPrefixBytes(key)
	iter := k.store.Prefix(ctx, keyBytes, store.Unlimited)
	if iter.Err() != nil {
		return nil, fmt.Errorf("querying accounts for key %q: %w", key, iter.Err())
	}
	for iter.Next() {
		item := iter.Item()
		_, keyBlockNum := Keys.unpack(item.Key)
		if keyBlockNum <= atBlockNum {
			accounts := decodeAccounts(item.Value)
			return accounts, nil
		}
	}

	return nil, nil
}

func decodeAccounts(payload []byte) Accounts {
	var accounts Accounts
	for i := 0; i < len(payload); i += 32 {
		accounts = append(accounts, payload[i:i+32])
	}
	return accounts
}

func encodeAccounts(accounts Accounts) []byte {
	var payload []byte

	for _, account := range accounts {
		payload = append(payload, []byte(account)...)
	}
	return payload
}

type mockBlockAccount struct {
	blockNum int64
	accounts Accounts
}

type mockAccountsStore struct {
	tables map[string][]*mockBlockAccount
}

type mockAccountsResolver struct {
	accountsStore *mockAccountsStore
}

func newMockAccountsStore() *mockAccountsStore {
	tables := make(map[string][]*mockBlockAccount)
	return &mockAccountsStore{
		tables: tables,
	}
}

func (m *mockAccountsResolver) Extended(blockNum int64, key Account, accounts Accounts) error {
	if accountBlocks, found := m.accountsStore.tables[key.base58()]; found {
		ab := accountBlocks[len(accountBlocks)-1]
		newAccountBlock := &mockBlockAccount{
			blockNum: blockNum,
			accounts: append(ab.accounts, accounts...),
		}
		accountBlocks = append(accountBlocks, newAccountBlock)
	}
	return nil
}

func (m *mockAccountsResolver) Resolve(blockNum int64, key Account) (Accounts, error) {
	accountBlocks := m.accountsStore.tables[key.base58()]
	for _, ab := range accountBlocks {
		if ab.blockNum >= blockNum {
			return ab.accounts, nil
		}
	}
	return nil, nil
}
