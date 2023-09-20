package accountsresolver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/streamingfast/kvdb/store"
)

type AccountsResolver interface {
	Extend(ctx context.Context, blockNum uint64, trxHash []byte, key Account, accounts Accounts) error
	Resolve(ctx context.Context, atBlockNum uint64, key Account) (Accounts, bool, error)
	StoreCursor(ctx context.Context, readerName string, cursor *Cursor) error
	GetCursor(ctx context.Context, readerName string) (*Cursor, error)
}

type cacheItem struct {
	blockNum uint64
	accounts Accounts
}

type KVDBAccountsResolver struct {
	store store.KVStore
	cache map[string][]*cacheItem
}

func NewKVDBAccountsResolver(store store.KVStore) *KVDBAccountsResolver {
	return &KVDBAccountsResolver{
		store: store,
		cache: make(map[string][]*cacheItem),
	}
}

func (r *KVDBAccountsResolver) Extend(ctx context.Context, blockNum uint64, trxHash []byte, key Account, accounts Accounts) error {
	if !r.isKnownTransaction(ctx, trxHash) {
		return nil
	}

	currentAccounts, _, err := r.Resolve(ctx, blockNum, key)
	if err != nil {
		return fmt.Errorf("retreiving last accounts for key %q: %w", key, err)
	}
	extendedAccount := append(currentAccounts, accounts...)
	payload := encodeAccounts(extendedAccount)
	err = r.store.Put(ctx, Keys.extendTableLookup(key, blockNum), payload)
	if err != nil {
		return fmt.Errorf("writing extended accounts for key %q: %w", key, err)
	}

	err = r.store.Put(ctx, Keys.knownTransaction(trxHash), []byte{})
	if err != nil {
		return fmt.Errorf("writing known transaction %x: %w", trxHash, err)
	}
	err = r.store.FlushPuts(ctx) //todo: move that up in call stack
	if err != nil {
		return fmt.Errorf("flushing extended accounts for key %q: %w", key, err)
	}

	r.cache[key.base58()] = append([]*cacheItem{{
		blockNum: blockNum,
		accounts: extendedAccount,
	}}, r.cache[key.base58()]...)

	return nil
}

func (r *KVDBAccountsResolver) Resolve(ctx context.Context, atBlockNum uint64, key Account) (Accounts, bool, error) {
	if cacheItems, ok := r.cache[key.base58()]; ok {
		for _, cacheItem := range cacheItems {
			if cacheItem.blockNum <= atBlockNum {
				return cacheItem.accounts, true, nil
			}
		}
	}

	keyBytes := Keys.tableLookupPrefix(key)
	iter := r.store.Prefix(ctx, keyBytes, store.Unlimited)
	if iter.Err() != nil {
		return nil, false, fmt.Errorf("querying accounts for key %q: %w", key, iter.Err())
	}

	for iter.Next() {
		item := iter.Item()
		_, keyBlockNum := Keys.unpackTableLookup(item.Key)
		if keyBlockNum <= atBlockNum {
			return decodeAccounts(item.Value), false, nil
		}
	}

	return nil, false, nil
}

func (r *KVDBAccountsResolver) isKnownTransaction(ctx context.Context, transactionHash []byte) bool {
	trxKey := Keys.knownTransaction(transactionHash)
	_, err := r.store.Get(ctx, trxKey)
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
