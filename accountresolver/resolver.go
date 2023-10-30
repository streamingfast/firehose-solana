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
	store  store.KVStore
	cache  map[string][]*cacheItem
	logger *zap.Logger
}

func NewKVDBAccountsResolver(store store.KVStore, logger *zap.Logger) *KVDBAccountsResolver {
	return &KVDBAccountsResolver{
		store:  store,
		cache:  make(map[string][]*cacheItem),
		logger: logger,
	}
}

func (r *KVDBAccountsResolver) CreateOrDelete(ctx context.Context, blockNum uint64, trxHash []byte, instructionIndex string, key Account) error {
	err := r.store.Put(ctx, Keys.extendTableLookup(key, blockNum), nil)
	if err != nil {
		return fmt.Errorf("reseting table account for %s: %w", key, err)
	}

	err = r.store.FlushPuts(ctx)
	if err != nil {
		return fmt.Errorf("flushing extended accounts for key %q: %w", key, err)
	}

	r.pushToCache(blockNum, key.Base58(), nil)

	return nil
}

func (r *KVDBAccountsResolver) Extend(ctx context.Context, blockNum uint64, trxHash []byte, instructionIndex string, key Account, accounts Accounts) error {
	currentAccounts, _, err := r.Resolve(ctx, blockNum, key)
	if err != nil {
		return fmt.Errorf("retrieving last accounts for key %q: %w", key, err)
	}
	extendedAccounts := append(currentAccounts, accounts...)
	payload := encodeAccounts(extendedAccounts)
	err = r.store.Put(ctx, Keys.extendTableLookup(key, blockNum), payload)
	if err != nil {
		return fmt.Errorf("writing extended accounts for key %q: %w", key, err)
	}

	err = r.store.Put(ctx, Keys.knownInstruction(trxHash, instructionIndex), []byte{})
	if err != nil {
		return fmt.Errorf("writing known transaction %x: %w", trxHash, err)
	}
	err = r.store.FlushPuts(ctx) //todo: move that up in call stack
	if err != nil {
		return fmt.Errorf("flushing extended accounts for key %q: %w", key, err)
	}

	r.pushToCache(blockNum, key.Base58(), extendedAccounts)

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

		r.pushToCache(atBlockNum, key.Base58(), resolvedAccounts)

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

		r.pushToCache(keyBlockNum, key.Base58(), accounts)

		if keyBlockNum <= atBlockNum && resolvedAccounts == nil {
			resolvedAccounts = accounts
		}
	}

	if iter.Err() != nil {
		return nil, 0, false, fmt.Errorf("querying accounts for key %q: %w", key, iter.Err())
	}

	return resolvedAccounts, keyBlockNum, false, nil
}

func (r *KVDBAccountsResolver) isKnownInstruction(ctx context.Context, transactionHash []byte, instructionIndex string) bool {
	instructionKey := Keys.knownInstruction(transactionHash, instructionIndex)
	_, err := r.store.Get(ctx, instructionKey)
	return errors.Is(err, store.ErrNotFound)
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

func (r *KVDBAccountsResolver) pushToCache(blockNum uint64, key string, accounts Accounts) {
	if cacheItems, found := r.cache[key]; found {
		for _, ci := range cacheItems {
			if ci.blockNum == blockNum {
				ci.accounts = append(ci.accounts, accounts...)
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
