package solana_accounts_resolver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/streamingfast/kvdb/store"
)

type AccountsResolver interface {
	Extended(ctx context.Context, blockNum uint64, key Account, accounts Accounts) error
	Resolve(ctx context.Context, blockNum uint64, key Account) (Accounts, int64, error)
	StoreCursor(ctx context.Context, cursor *Cursor) error
	GetCursor(ctx context.Context) (uint64, []byte, error)
}

type KVDBAccountsResolver struct {
	store store.KVStore
}

func NewKVDBAccountsResolver(store store.KVStore) *KVDBAccountsResolver {
	return &KVDBAccountsResolver{
		store: store,
	}
}

func (r *KVDBAccountsResolver) Extended(ctx context.Context, blockNum uint64, key Account, accounts Accounts) error {
	currentAccounts, resolveAtBlockNum, err := r.Resolve(ctx, blockNum, key)
	if err != nil {
		return fmt.Errorf("retreiving last accounts for key %q: %w", key, err)
	}

	if resolveAtBlockNum == blockNum {
		// already extended at this block, nothing to do
		return nil
	}

	payload := encodeAccounts(append(currentAccounts, accounts...))
	err = r.store.Put(ctx, Keys.extendTableLookup(key, blockNum), payload)
	if err != nil {
		return fmt.Errorf("writing extended accounts for key %q: %w", key, err)
	}
	err = r.store.FlushPuts(ctx)
	if err != nil {
		return fmt.Errorf("flushing extended accounts for key %q: %w", key, err)
	}

	return nil
}

func (r *KVDBAccountsResolver) Resolve(ctx context.Context, atBlockNum uint64, key Account) (Accounts, uint64, error) {
	keyBytes := Keys.tableLookupPrefix(key)
	iter := r.store.Prefix(ctx, keyBytes, store.Unlimited)
	if iter.Err() != nil {
		return nil, 0, fmt.Errorf("querying accounts for key %q: %w", key, iter.Err())
	}
	for iter.Next() {
		item := iter.Item()
		_, keyBlockNum := Keys.unpackTableLookup(item.Key)
		if keyBlockNum <= atBlockNum {
			accounts := decodeAccounts(item.Value)
			return accounts, keyBlockNum, nil
		}
	}

	return nil, 0, nil
}

func (r *KVDBAccountsResolver) StoreCursor(ctx context.Context, readerName string, cursor *Cursor) error {
	payload := make([]byte, 8+32)
	binary.BigEndian.PutUint64(payload[:8], cursor.blockNum)
	copy(payload[8:], cursor.blockHash)
	err := r.store.Put(ctx, Keys.cursor(readerName), payload)
	if err != nil {
		return fmt.Errorf("writing cursor: %w", err)
	}

	err = r.store.FlushPuts(ctx)
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
	blockNum := binary.BigEndian.Uint64(payload[:8])
	blockHash := payload[8:]
	return newCursor(blockNum, blockHash), nil
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
