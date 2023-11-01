package accountsresolver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/streamingfast/kvdb/store"
	"go.uber.org/zap"
)

type cacheItem struct {
	blockNum uint64
	accounts Accounts
}

type KVDBAccountsResolver struct {
	store    store.KVStore
	cache    map[string][]*cacheItem
	logger   *zap.Logger
	toCommit map[string][]Account
}

func NewKVDBAccountsResolver(store store.KVStore, logger *zap.Logger) *KVDBAccountsResolver {
	return &KVDBAccountsResolver{
		store:    store,
		cache:    make(map[string][]*cacheItem),
		toCommit: make(map[string][]Account),
		logger:   logger,
	}
}

func (r *KVDBAccountsResolver) CreateOrDelete(key Account) error {
	r.toCommit[key.Base58()] = nil
	return nil
}

func (r *KVDBAccountsResolver) CommitBlock(ctx context.Context, blockNum uint64) error {
	for base58key, accounts := range r.toCommit {
		tableKey := MustFromBase58(base58key)
		currentAccounts, _, err := r.Resolve(ctx, blockNum, tableKey)
		if err != nil {
			return fmt.Errorf("retrieving last accounts for tableKey %q: %w", tableKey, err)
		}

		var payload []byte
		var extendedAccounts Accounts
		if accounts != nil { // nil means delete or create
			extendedAccounts = append(currentAccounts, accounts...)
			payload = encodeAccounts(extendedAccounts)

		}

		err = r.store.Put(ctx, Keys.extendTableLookup(tableKey, blockNum), payload)
		if err != nil {
			return fmt.Errorf("writing extended accounts for tableKey %q: %w", tableKey, err)
		}

		err = r.store.FlushPuts(ctx)
		if err != nil {
			return fmt.Errorf("flushing extended accounts for tableKey %q: %w", tableKey, err)
		}

		r.prependToCache(blockNum, tableKey.Base58(), extendedAccounts)
	}
	r.toCommit = make(map[string][]Account)
	return nil
}

func (r *KVDBAccountsResolver) Extend(key Account, accounts Accounts) error {
	r.toCommit[key.Base58()] = append(r.toCommit[key.Base58()], accounts...)
	return nil
}

func (r *KVDBAccountsResolver) Resolve(ctx context.Context, atBlockNum uint64, key Account) (Accounts, bool, error) {
	if cacheItems, ok := r.cache[key.Base58()]; ok {
		for _, cacheItem := range cacheItems {
			if cacheItem.blockNum < atBlockNum {
				return cacheItem.accounts, true, nil
			}
		}
	}

	keyBytes := Keys.tableLookupPrefix(key)
	iter := r.store.Prefix(ctx, keyBytes, store.Unlimited)

	var resolvedAccounts Accounts
	for iter.Next() {
		item := iter.Item()
		_, keyBlockNum := Keys.UnpackTableLookup(item.Key)
		accounts := DecodeAccounts(item.Value)

		r.appendToCache(keyBlockNum, key.Base58(), accounts)

		if keyBlockNum < atBlockNum && resolvedAccounts == nil {
			resolvedAccounts = accounts
		}
	}

	if iter.Err() != nil {
		return nil, false, fmt.Errorf("querying accounts for key %q: %w", key, iter.Err())
	}

	return resolvedAccounts, false, nil
}

func (r *KVDBAccountsResolver) ResolveWithBlock(ctx context.Context, atBlockNum uint64, key Account) (Accounts, uint64, bool, error) {
	if cacheItems, ok := r.cache[key.Base58()]; ok {
		for _, cacheItem := range cacheItems {
			if cacheItem.blockNum < atBlockNum {
				return cacheItem.accounts, cacheItem.blockNum, true, nil
			}
		}
	}

	keyBytes := Keys.tableLookupPrefix(key)
	iter := r.store.Prefix(ctx, keyBytes, store.Unlimited)

	var resolvedAccounts Accounts
	keyBlockNum := uint64(0)
	for iter.Next() {
		item := iter.Item()
		_, keyBlockNum = Keys.UnpackTableLookup(item.Key)
		accounts := DecodeAccounts(item.Value)

		r.appendToCache(keyBlockNum, key.Base58(), accounts)

		if keyBlockNum <= atBlockNum && resolvedAccounts == nil {
			resolvedAccounts = accounts
		}
	}

	if iter.Err() != nil {
		return nil, 0, false, fmt.Errorf("querying accounts for key %q: %w", key, iter.Err())
	}

	return resolvedAccounts, keyBlockNum, false, nil
}

func (r *KVDBAccountsResolver) StoreCursor(ctx context.Context, readerName string, cursor *Cursor) error {
	payload := make([]byte, 8+32)
	binary.BigEndian.PutUint64(payload, cursor.slotNum)
	err := r.store.Put(ctx, Keys.cursor(readerName), payload)
	if err != nil {
		return fmt.Errorf("writing cursor: %w", err)
	}

	err = r.store.FlushPuts(ctx) //todo: move that up in call stack
	if err != nil {
		return fmt.Errorf("flushing cursor: %w", err)
	}
	return nil
}

func (r *KVDBAccountsResolver) GetCursor(ctx context.Context, readerName string) (*Cursor, error) {
	payload, err := r.store.Get(ctx, Keys.cursor(readerName))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting cursor: %w", err)
	}
	if payload == nil {
		return nil, nil
	}
	blockNum := binary.BigEndian.Uint64(payload)
	return NewCursor(blockNum), nil
}

func (r *KVDBAccountsResolver) appendToCache(blockNum uint64, key string, accounts Accounts) {
	if cacheItems, found := r.cache[key]; found {
		for _, ci := range cacheItems {
			if ci.blockNum == blockNum {
				return
			}
		}
	}
	r.cache[key] = append(r.cache[key], []*cacheItem{
		{
			blockNum: blockNum,
			accounts: accounts,
		},
	}...)
}
func (r *KVDBAccountsResolver) prependToCache(blockNum uint64, key string, accounts Accounts) {
	if cacheItems, found := r.cache[key]; found {
		for _, ci := range cacheItems {
			if ci.blockNum == blockNum {
				return
			}
		}
	}
	r.cache[key] = append([]*cacheItem{
		{
			blockNum: blockNum,
			accounts: accounts,
		},
	}, r.cache[key]...)
}

func DecodeAccounts(payload []byte) Accounts {
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
