package accountsresolver

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/streamingfast/kvdb/store"
)

type AccountsResolver interface {
	Extend(ctx context.Context, blockNum uint64, trxHash []byte, key Account, accounts Accounts) error
	Resolve(ctx context.Context, atBlockNum uint64, key Account) (Accounts, uint64, []byte, error)
	ResolveSeenTransaction(ctx context.Context, atBlockNum uint64, trxHash []byte) ([]byte, error)
	SeenTransaction(ctx context.Context, atBlockNum uint64, trxHash []byte) error
	StoreCursor(ctx context.Context, readerName string, cursor *Cursor) error
	GetCursor(ctx context.Context, readerName string) (*Cursor, error)
}

type KVDBAccountsResolver struct {
	store store.KVStore
}

func NewKVDBAccountsResolver(store store.KVStore) *KVDBAccountsResolver {
	return &KVDBAccountsResolver{
		store: store,
	}
}

func (r *KVDBAccountsResolver) Extend(ctx context.Context, blockNum uint64, trxHash []byte, key Account, accounts Accounts) error {
	currentAccounts, resolveAtBlockNum, keyTrxHash, err := r.Resolve(ctx, blockNum, key)
	if err != nil {
		return fmt.Errorf("retreiving last accounts for key %q: %w", key, err)
	}

	if resolveAtBlockNum == blockNum && bytes.Equal(trxHash, keyTrxHash) {
		// already extended at this block, nothing to do
		return nil
	}

	payload := encodeAccounts(append(currentAccounts, accounts...))
	err = r.store.Put(ctx, Keys.extendTableLookup(key, blockNum, trxHash), payload)
	if err != nil {
		return fmt.Errorf("writing extended accounts for key %q: %w", key, err)
	}
	if err != nil {
		return fmt.Errorf("flushing extended accounts for key %q: %w", key, err)
	}

	return nil
}

func (r *KVDBAccountsResolver) Resolve(ctx context.Context, atBlockNum uint64, key Account) (Accounts, uint64, []byte, error) {
	keyBytes := Keys.tableLookupPrefix(key)
	iter := r.store.Prefix(ctx, keyBytes, store.Unlimited)
	if iter.Err() != nil {
		return nil, 0, nil, fmt.Errorf("querying accounts for key %q: %w", key, iter.Err())
	}

	for iter.Next() {
		item := iter.Item()
		_, keyBlockNum, hash := Keys.unpackTableLookup(item.Key)
		if keyBlockNum <= atBlockNum {
			return decodeAccounts(item.Value), keyBlockNum, hash, nil
		}
	}

	return nil, 0, nil, nil
}

func (r *KVDBAccountsResolver) ResolveSeenTransaction(ctx context.Context, atBlockNum uint64, trxHash []byte) ([]byte, error) {
	val, err := r.store.Get(ctx, Keys.transactionSeen(atBlockNum, trxHash))
	if err != nil {

	}
	if val != nil {

	}
}

func (r *KVDBAccountsResolver) SeenTransaction(ctx context.Context, atBlockNum uint64, trxHash []byte) error {

	return nil
	//err := r.store.Put()
	//
	//return nil, nil
}

func (r *KVDBAccountsResolver) StoreCursor(ctx context.Context, readerName string, cursor *Cursor) error {
	payload := make([]byte, 8+32)
	binary.BigEndian.PutUint64(payload, cursor.slotNum)
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
