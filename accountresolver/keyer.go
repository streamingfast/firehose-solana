package solana_accounts_resolver

import (
	"encoding/binary"
	"math"
)

var Keys keyer

type keyer struct{}

func (keyer) extendedKeyBytes(key Account, blockNum uint64) []byte {
	keyBytes := []byte(key)
	return append(keyBytes, revBlockNumBytes(blockNum)...)
}

func (keyer) lookupKeyBytes(key Account) []byte {
	return []byte(key)
}

func (keyer) unpack(key []byte) (Account, uint64) {
	return Account(key[:32]), binary.BigEndian.Uint64(key[32:])
}

func revBlockNumBytes(blockNum uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, math.MaxUint64-blockNum)
	return bytes
}
