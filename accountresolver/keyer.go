package accountsresolver

import (
	"encoding/binary"
	"math"
)

// al:table_address:reader_1:block = []
// al:table_address:block = accounts

// cur:reader_1:block = []

const tableAccountLookup = 0x0
const tableCursor = 0x1
const tableKnownTransaction = 0x3

var Keys keyer

type keyer struct{}

func (keyer) extendTableLookup(key Account, blockNum uint64) (out []byte) {
	out = make([]byte, 1+32+8)
	out[0] = tableAccountLookup
	copy(out[1:33], key)
	binary.BigEndian.PutUint64(out[33:41], math.MaxUint64-blockNum)
	return out
}

func (keyer) tableLookupPrefix(key Account) (out []byte) {
	out = make([]byte, 1+32)
	out[0] = tableAccountLookup
	copy(out[1:33], key)
	return out
}

func (keyer) unpackTableLookup(key []byte) (Account, uint64) {
	return key[1:33], math.MaxUint64 - binary.BigEndian.Uint64(key[33:])
}

func (keyer) cursor(readerName string) (out []byte) {
	out = make([]byte, len(readerName)+1)
	out[0] = tableCursor
	copy(out[1:], readerName)
	return out
}

func (keyer) transactionSeenPrefix(blockNum uint64) []byte {
	out := make([]byte, 1+9)
	out[0] = tableKnownTransaction
	binary.BigEndian.PutUint64(out[1:], math.MaxUint64-blockNum)
	return out
}

func (keyer) knownTransaction(trxHash []byte) []byte {
	out := make([]byte, 1+64)
	out[0] = tableKnownTransaction
	copy(out[1:], trxHash)
	return out
}

func (keyer) unpackKnownTransaction(key []byte) (trxHash []byte) {
	return key[1:]
}