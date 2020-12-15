package keyer

import (
	"encoding/binary"

	"github.com/dfuse-io/solana-go"
)

const (
	PrefixFillData = byte(0x01)

	PrefixFillsByPubkey       = byte(0x02)
	PrefixFillsByMarketPubkey = byte(0x03)

	PrefixCheckpoint = byte(0x04)
)

//EventQueue:
//write:
//
//
//NewOrder:
//write:
//// to query all markets, for a pubkey
//
//// to query a single market for a given pubkey

//
//LastWrittenBlock:
//write:
//last_written_block => PB:slot_num:slot_id

// orders:[market]:[order_seq_num]:[slot_num] => FillData(side)
func EncodeFillData(market solana.PublicKey, orderSeqNum uint64, slotNumber uint64) []byte {

	key := make([]byte, 1+32+8+8)

	key[0] = PrefixFillData
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], orderSeqNum)
	binary.BigEndian.PutUint64(key[41:], slotNumber)
	return key
}

func DecodeFillData(key []byte) (market solana.PublicKey, orderSeqNum uint64, slotNum uint64) {
	copy(market[:], key[1:])
	orderSeqNum = binary.BigEndian.Uint64(key[33:])
	slotNum = binary.BigEndian.Uint64(key[41:])
	return
}

// order_pubkey:[pubkey]:[rev_slot_num]:[market]:[rev_order_seq_num] => nil
func EncodeFillsByPubkey(pubkey, market solana.PublicKey, orderSeqNum uint64, slotNumber uint64) []byte {
	key := make([]byte, 1+32+32+8+8)

	key[0] = PrefixFillsByPubkey
	copy(key[1:], pubkey[:])
	binary.BigEndian.PutUint64(key[33:], ^slotNumber)
	copy(key[41:], market[:])
	binary.BigEndian.PutUint64(key[73:], ^orderSeqNum)

	return key

}

func DecodeFillsByPubkey(key []byte) (pubkey, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) {
	copy(pubkey[:], key[1:])
	slotNum = ^binary.BigEndian.Uint64(key[33:])
	copy(market[:], key[41:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[73:])
	return
}

// order_market:[market]:[pubkey]:[rev_slot_num]:[rev_order_seq_num] => nil
func EncodeFillsByMarketPubkey(pubkey, market solana.PublicKey, orderSeqNum uint64, slotNumber uint64) []byte {
	key := make([]byte, 1+32+32+8+8)

	key[0] = PrefixFillsByMarketPubkey
	copy(key[1:], market[:])
	copy(key[33:], pubkey[:])
	binary.BigEndian.PutUint64(key[65:], ^slotNumber)
	binary.BigEndian.PutUint64(key[73:], ^orderSeqNum)

	return key

}

func DecodeFillsByMarketPubkey(key []byte) (pubkey, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) {
	copy(market[:], key[1:])
	copy(pubkey[:], key[33:])
	slotNum = ^binary.BigEndian.Uint64(key[65:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[73:])
	return
}

func EncodeCheckpoint() []byte {
	return []byte{PrefixCheckpoint, 0x6c, 0x61, 0x73, 0x74} // last in ascii

}