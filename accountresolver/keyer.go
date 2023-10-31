package accountsresolver

import (
	"encoding/binary"
	"math"
)

const TableAccountLookup = 0x0
const TableCursor = 0x1

var Keys keyer

type keyer struct{}

func (keyer) extendTableLookup(key Account, blockNum uint64) (out []byte) {
	out = make([]byte, 1+32+8)
	out[0] = TableAccountLookup
	copy(out[1:33], key)
	binary.BigEndian.PutUint64(out[33:41], math.MaxUint64-blockNum)
	return out
}

func (keyer) tableLookupPrefix(key Account) (out []byte) {
	out = make([]byte, 1+32)
	out[0] = TableAccountLookup
	copy(out[1:33], key)
	return out
}

func (keyer) UnpackTableLookup(key []byte) (Account, uint64) {
	return key[1:33], math.MaxUint64 - binary.BigEndian.Uint64(key[33:])
}

func (keyer) cursor(readerName string) (out []byte) {
	out = make([]byte, len(readerName)+1)
	out[0] = TableCursor
	copy(out[1:], readerName)
	return out
}
