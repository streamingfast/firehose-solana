package solana_accounts_resolver

import (
	"encoding/binary"
	"math"
)

const tableAccountLookup = 0x0
const tableCursor = 0x1

var Keys keyer

type keyer struct{}

func (keyer) extendTableLookup(key Account, blockNum uint64) (out []byte) {
	out = make([]byte, 1+32+8)
	out[0] = tableAccountLookup
	copy(out[1:33], key)
	binary.BigEndian.PutUint64(out[33:], math.MaxUint64-blockNum)
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

func (keyer) cursor() (out []byte) {
	out = make([]byte, 1)
	out[0] = tableCursor
	return out
}
