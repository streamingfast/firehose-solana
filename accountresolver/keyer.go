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

var Keys keyer

type keyer struct{}

func (keyer) extendTableLookup(key Account, blockNum uint64, trxHash []byte) (out []byte) {
	out = make([]byte, 1+32+8+64)
	out[0] = tableAccountLookup
	copy(out[1:33], key)
	binary.BigEndian.PutUint64(out[33:41], math.MaxUint64-blockNum)
	copy(out[41:], trxHash)
	return out
}

func (keyer) tableLookupPrefix(key Account) (out []byte) {
	out = make([]byte, 1+32)
	out[0] = tableAccountLookup
	copy(out[1:33], key)
	return out
}

func (keyer) unpackTableLookup(key []byte) (Account, uint64, []byte) {
	return key[1:33], math.MaxUint64 - binary.BigEndian.Uint64(key[33:41]), key[41:]
}

func (keyer) cursor(readerName string) (out []byte) {
	out = make([]byte, len(readerName)+1)
	out[0] = tableCursor
	copy(out[1:], readerName)

	return out
}
