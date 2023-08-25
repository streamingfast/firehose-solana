package solana_accounts_resolver

import (
	"encoding/binary"
	"encoding/hex"
	"math"
)

var Keys keyer

type keyer struct{}

func (keyer) extendedKeyBytes(key Account, blockNum uint64) (out []byte) {
	keyBytes := []byte(key)
	out = append(keyBytes, revBlockNumBytes(blockNum)...)
	println("extendedKeyBytes\t", hex.EncodeToString(out))
	return out
	//return Keys.lookupPrefixBytes(key)
}

func (keyer) lookupPrefixBytes(key Account) (out []byte) {
	out = key
	println("lookupPrefixBytes\t\t", hex.EncodeToString(out))
	return out
}

func (keyer) unpack(key []byte) (Account, uint64) {
	return Account(key[:32]), math.MaxUint64 - binary.BigEndian.Uint64(key[32:])
}

func revBlockNumBytes(blockNum uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, math.MaxUint64-blockNum)
	return bytes
}
