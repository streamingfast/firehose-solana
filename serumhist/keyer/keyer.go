package keyer

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"go.uber.org/zap"

	"github.com/dfuse-io/solana-go"
)

const (
	PrefixFillData = byte(0x01)

	PrefixOrdersByPubkey       = byte(0x02)
	PrefixOrdersByMarketPubkey = byte(0x03)

	PrefixCheckpoint = byte(0x04)
)

type Prefix []byte

func (p Prefix) String() string { return hex.EncodeToString(p) }

type Key []byte

func (k Key) String() string { return hex.EncodeToString(k) }

// orders:[market]:[order_seq_num]:[slot_num] => FillData(side)
func EncodeFillData(market solana.PublicKey, orderSeqNum uint64, slotNum uint64) Key {
	key := make([]byte, 1+32+8+8)

	key[0] = PrefixFillData
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], orderSeqNum)
	binary.BigEndian.PutUint64(key[41:], slotNum)

	if traceEnabled {
		zlog.Debug("fill data encoded",
			zap.Stringer("market", market),
			zap.Uint64("order_seq_num", orderSeqNum),
			zap.Uint64("slot_num", slotNum),
			zap.Stringer("prefix", Key(key)))
	}

	return key
}

func EncodedPrefixFillData(market solana.PublicKey, orderSeqNum uint64) Prefix {
	key := make([]byte, 1+32+8)

	key[0] = PrefixFillData
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], orderSeqNum)

	if traceEnabled {
		zlog.Debug("fill data prefix encoded",
			zap.Stringer("market", market),
			zap.Uint64("order_seq_num", orderSeqNum),
			zap.Stringer("prefix", Key(key)))
	}

	return key
}

func DecodeFillData(key Key) (market solana.PublicKey, orderSeqNum uint64, slotNum uint64) {
	if key[0] != PrefixFillData {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixFillData))
	}
	copy(market[:], key[1:])
	orderSeqNum = binary.BigEndian.Uint64(key[33:])
	slotNum = binary.BigEndian.Uint64(key[41:])

	if traceEnabled {
		zlog.Debug("fill data key decoded",
			zap.Stringer("market", market),
			zap.Uint64("order_seq_num", orderSeqNum),
			zap.Uint64("slot_num", slotNum),
			zap.Stringer("key", key))
	}

	return
}

// order_pubkey:[pubkey]:[rev_slot_num]:[market]:[rev_order_seq_num] => nil
func EncodeOrdersByPubkey(trader, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) Key {
	key := make([]byte, 1+32+32+8+8)

	key[0] = PrefixOrdersByPubkey
	copy(key[1:], trader[:])
	binary.BigEndian.PutUint64(key[33:], ^slotNum)
	copy(key[41:], market[:])
	binary.BigEndian.PutUint64(key[73:], ^orderSeqNum)

	if traceEnabled {
		zlog.Debug("orders by pub key encoded",
			zap.Stringer("trader", trader),
			zap.Stringer("market", market),
			zap.Uint64("order_seq_num", orderSeqNum),
			zap.Uint64("slot_num", slotNum),
			zap.Stringer("key", Key(key)),
		)
	}

	return key
}

// order_pubkey:[pubkey]
func EncodeOrdersPrefixByPubkey(trader solana.PublicKey) Prefix {
	key := make([]byte, 1+32)

	key[0] = PrefixOrdersByPubkey
	copy(key[1:], trader[:])

	if traceEnabled {
		zlog.Debug("orders by pub  prefix encoded",
			zap.Stringer("trader", trader),
			zap.Stringer("key", Key(key)),
		)
	}

	return key
}

func DecodeOrdersByPubkey(key Key) (trader, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) {
	if key[0] != PrefixOrdersByPubkey {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixOrdersByPubkey))
	}
	copy(trader[:], key[1:])
	slotNum = ^binary.BigEndian.Uint64(key[33:])
	copy(market[:], key[41:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[73:])

	if traceEnabled {
		zlog.Debug("orders by pub key decoded",
			zap.Stringer("trader", trader),
			zap.Stringer("marker", market),
			zap.Uint64("order_seq_num", orderSeqNum),
			zap.Uint64("slot_num", slotNum),
			zap.Stringer("key", key),
		)
	}

	return
}

// order_market:[market]:[pubkey]:[rev_slot_num]:[rev_order_seq_num] => nil
func EncodeOrdersByMarketPubkey(trader, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) Key {
	key := make([]byte, 1+32+32+8+8)

	key[0] = PrefixOrdersByMarketPubkey
	copy(key[1:], market[:])
	copy(key[33:], trader[:])
	binary.BigEndian.PutUint64(key[65:], ^slotNum)
	binary.BigEndian.PutUint64(key[73:], ^orderSeqNum)

	if traceEnabled {
		zlog.Debug("orders by market pub key encoded",
			zap.Stringer("trader", trader),
			zap.Stringer("marker", market),
			zap.Uint64("order_seq_num", orderSeqNum),
			zap.Uint64("slot_num", slotNum),
			zap.Stringer("key", Key(key)),
		)
	}

	return key
}

// order_market:[market]:[pubkey]
func EncodeOrdersPrefixByMarketPubkey(trader, market solana.PublicKey) Prefix {
	key := make([]byte, 1+32+32)

	key[0] = PrefixOrdersByMarketPubkey
	copy(key[1:], market[:])
	copy(key[33:], trader[:])

	if traceEnabled {
		zlog.Debug("orders by market pub prefix encoded",
			zap.Stringer("trader", trader),
			zap.Stringer("marker", market),
			zap.Stringer("key", Key(key)),
		)
	}

	return key
}

func DecodeOrdersByMarketPubkey(key Key) (trader, market solana.PublicKey, orderSeqNum uint64, slotNum uint64) {
	if key[0] != PrefixOrdersByMarketPubkey {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixOrdersByMarketPubkey))
	}
	copy(market[:], key[1:])
	copy(trader[:], key[33:])
	slotNum = ^binary.BigEndian.Uint64(key[65:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[73:])

	if traceEnabled {
		zlog.Debug("orders by market pub key decoded",
			zap.Stringer("trader", trader),
			zap.Stringer("marker", market),
			zap.Uint64("order_seq_num", orderSeqNum),
			zap.Uint64("slot_num", slotNum),
			zap.Stringer("key", key),
		)
	}

	return
}

func EncodeCheckpoint() []byte {
	return []byte{PrefixCheckpoint, 0x6c, 0x61, 0x73, 0x74} // last in ascii

}
