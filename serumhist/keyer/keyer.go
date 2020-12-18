package keyer

import (
	"encoding/binary"

	"github.com/dfuse-io/solana-go"
)

const (
	PrefixFillData = byte(0x01)

	PrefixOrdersByPubkey       = byte(0x02)
	PrefixOrdersByMarketPubkey = byte(0x03)

	PrefixCheckpoint = byte(0x04)
)

// orders:[market]:[order_seq_num]:[slot_num] => FillData(side)
func EncodeFillData(market solana.PublicKey, orderSeqNum uint64, slotNum uint64) []byte {

	key := make([]byte, 1+32+8+8)

	key[0] = PrefixFillData
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], orderSeqNum)
	binary.BigEndian.PutUint64(key[41:], slotNum)
	return key
}

func DecodeFillData(key []byte) (market solana.PublicKey, orderSeqNum uint64, slotNum uint64) {
	copy(market[:], key[1:])
	orderSeqNum = binary.BigEndian.Uint64(key[33:])
	slotNum = binary.BigEndian.Uint64(key[41:])
	return
}

func EncodeGetFillData(market solana.PublicKey, orderSeqNum uint64) []byte {

	key := make([]byte, 1+32+8)

	key[0] = PrefixFillData
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], orderSeqNum)
	return key
}

// order_pubkey:[pubkey]:[rev_slot_num]:[market]:[rev_order_seq_num] => nil
func EncodeOrdersByPubkey(trader, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) []byte {
	key := make([]byte, 1+32+32+8+8)

	key[0] = PrefixOrdersByPubkey
	copy(key[1:], trader[:])
	binary.BigEndian.PutUint64(key[33:], ^slotNum)
	copy(key[41:], market[:])
	binary.BigEndian.PutUint64(key[73:], ^orderSeqNum)

	return key

}

func DecodeOrdersByPubkey(key []byte) (trader, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) {
	copy(trader[:], key[1:])
	slotNum = ^binary.BigEndian.Uint64(key[33:])
	copy(market[:], key[41:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[73:])
	return
}

// order_market:[market]:[pubkey]:[rev_slot_num]:[rev_order_seq_num] => nil
func EncodeOrdersByMarketPubkey(trader, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) []byte {
	key := make([]byte, 1+32+32+8+8)

	key[0] = PrefixOrdersByMarketPubkey
	copy(key[1:], market[:])
	copy(key[33:], trader[:])
	binary.BigEndian.PutUint64(key[65:], ^slotNum)
	binary.BigEndian.PutUint64(key[73:], ^orderSeqNum)

	return key

}

// order_market:[market]:[pubkey]
func EncodeOrdersPrefixByMarketPubkey(trader, market solana.PublicKey) []byte {
	key := make([]byte, 1+32+32)

	key[0] = PrefixOrdersByMarketPubkey
	copy(key[1:], market[:])
	copy(key[33:], trader[:])

	return key

}

func DecodeOrdersByMarketPubkey(key []byte) (trader, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) {
	copy(market[:], key[1:])
	copy(trader[:], key[33:])
	slotNum = ^binary.BigEndian.Uint64(key[65:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[73:])
	return
}

func EncodeCheckpoint() []byte {
	return []byte{PrefixCheckpoint, 0x6c, 0x61, 0x73, 0x74} // last in ascii

}
