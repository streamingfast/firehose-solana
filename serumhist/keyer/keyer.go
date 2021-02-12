package keyer

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/solana-go"
)

// note: the trader, trx_index, inst_index and market are not marshalled in the proto Fill or Order, this is why we need to them in the keys to augment
// the proto def
const (
	PrefixFillByTrader       = byte(0x01) // 01:[trader]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[market]:[rev_order_seq_num] 	=>  FillData(side)
	PrefixFillByTraderMarket = byte(0x02) // 02:[trader]:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[rev_order_seq_num] 	=>  FillData(side)
	PrefixFillByMarket       = byte(0x03) // 03:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[trader]:[rev_order_seq_num] 	=> FillData(side)
	PrefixFillByOrder        = byte(0x04) // 04:[market]:[rev_order_seq_num]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[trader]	=> FillData(side)

	PrefixTradingAccount = byte(0x05) // 05:[trading_account] => [trader]

	PrefixOrderByMarket       = byte(0x06) // 06:[market]:[rev_order_seq_num]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[trader] => Order
	PrefixOrderByTrader       = byte(0x07) // 07:[trader]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[market]:[rev_order_seq_num] => Pointer ot 06 key
	PrefixOrderByTraderMarket = byte(0x08) // 08:[trader]:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[rev_order_seq_num] => Pointer ot 06 key

	PrefixCheckpoint = byte(0x10)
)

// 01:[trader]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[market]:[rev_order_seq_num] =>  FillData(side)
func EncodeFillByTrader(trader, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+8+8+8+32+8)
	key[0] = PrefixFillByTrader
	copy(key[1:], trader[:])
	binary.BigEndian.PutUint64(key[33:], ^slotNum)
	binary.BigEndian.PutUint64(key[41:], ^trxIdx)
	binary.BigEndian.PutUint64(key[49:], ^instIdx)
	copy(key[57:], market[:])
	binary.BigEndian.PutUint64(key[89:], ^orderSeqNum)
	return key
}

func DecodeFillByTrader(key Key) (trader solana.PublicKey, market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
	if key[0] != PrefixFillByTrader {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixFillByTrader))
	}
	copy(trader[:], key[1:])
	slotNum = ^binary.BigEndian.Uint64(key[33:])
	trxIdx = ^binary.BigEndian.Uint64(key[41:])
	instIdx = ^binary.BigEndian.Uint64(key[49:])
	copy(market[:], key[57:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[89:])

	return
}

func EncodeFillByTraderPrefix(trader solana.PublicKey) Prefix {
	key := make([]byte, 1+32)
	key[0] = PrefixFillByTrader
	copy(key[1:], trader[:])
	return key
}

// 02:[trader]:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[rev_order_seq_num] =>  =>
func EncodeFillByMarketTrader(trader, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+32+8+8+8+8)
	key[0] = PrefixFillByTraderMarket
	copy(key[1:], trader[:])
	copy(key[33:], market[:])
	binary.BigEndian.PutUint64(key[65:], ^slotNum)
	binary.BigEndian.PutUint64(key[73:], ^trxIdx)
	binary.BigEndian.PutUint64(key[81:], ^instIdx)
	binary.BigEndian.PutUint64(key[89:], ^orderSeqNum)
	return key
}

func DecodeFillByMarketTrader(key Key) (trader solana.PublicKey, market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
	if key[0] != PrefixFillByTraderMarket {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixFillByTraderMarket))
	}
	copy(trader[:], key[1:])
	copy(market[:], key[33:])
	slotNum = ^binary.BigEndian.Uint64(key[65:])
	trxIdx = ^binary.BigEndian.Uint64(key[73:])
	instIdx = ^binary.BigEndian.Uint64(key[81:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[89:])
	return
}

func EncodeFillByTraderMarketPrefix(trader, market solana.PublicKey) Prefix {
	key := make([]byte, 1+32+32)
	key[0] = PrefixFillByTraderMarket
	copy(key[1:], trader[:])
	copy(key[33:], market[:])
	return key
}

// 03:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[trader]:[rev_order_seq_num] => FillData(side)
func EncodeFillByMarket(trader, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+8+8+8+32+8)
	key[0] = PrefixFillByMarket
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], ^slotNum)
	binary.BigEndian.PutUint64(key[41:], ^trxIdx)
	binary.BigEndian.PutUint64(key[49:], ^instIdx)
	copy(key[57:], trader[:])
	binary.BigEndian.PutUint64(key[89:], ^orderSeqNum)
	return key
}

func DecodeFillByMarket(key Key) (trader solana.PublicKey, market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
	if key[0] != PrefixFillByMarket {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixFillByMarket))
	}
	copy(market[:], key[1:])
	slotNum = ^binary.BigEndian.Uint64(key[33:])
	trxIdx = ^binary.BigEndian.Uint64(key[41:])
	instIdx = ^binary.BigEndian.Uint64(key[49:])
	copy(trader[:], key[57:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[89:])

	return
}

func EncodeFillByMarketPrefix(market solana.PublicKey) Prefix {
	key := make([]byte, 1+32)
	key[0] = PrefixFillByMarket
	copy(key[1:], market[:])
	return key
}

// 04:[market]:[rev_order_seq_num]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[trader]	=> FillData(side)
func EncodeFillByOrder(trader, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+8+8+8+8+32)
	key[0] = PrefixFillByOrder
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], ^orderSeqNum)
	binary.BigEndian.PutUint64(key[41:], ^slotNum)
	binary.BigEndian.PutUint64(key[49:], ^trxIdx)
	binary.BigEndian.PutUint64(key[57:], ^instIdx)
	copy(key[65:], trader[:])
	return key
}

func DecodeFillByOrder(key Key) (trader solana.PublicKey, market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
	if key[0] != PrefixFillByOrder {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixFillByOrder))
	}
	copy(market[:], key[1:])
	slotNum = ^binary.BigEndian.Uint64(key[33:])
	trxIdx = ^binary.BigEndian.Uint64(key[41:])
	instIdx = ^binary.BigEndian.Uint64(key[49:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[57:])
	copy(trader[:], key[65:])

	return
}

// 07:[trader]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[market]:[rev_order_seq_num] => Pointer ot 06 key
func EncodeOrderByTrader(trader, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+8+8+8+32+8)
	key[0] = PrefixOrderByTrader
	copy(key[1:], trader[:])
	binary.BigEndian.PutUint64(key[33:], ^slotNum)
	binary.BigEndian.PutUint64(key[41:], ^trxIdx)
	binary.BigEndian.PutUint64(key[49:], ^instIdx)
	copy(key[57:], market[:])
	binary.BigEndian.PutUint64(key[89:], ^orderSeqNum)
	return key
}

func DecodeOrderByTrader(key Key) (trader solana.PublicKey, market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
	if key[0] != PrefixOrderByTrader {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixOrderByTrader))
	}
	copy(trader[:], key[1:])
	slotNum = ^binary.BigEndian.Uint64(key[33:])
	trxIdx = ^binary.BigEndian.Uint64(key[41:])
	instIdx = ^binary.BigEndian.Uint64(key[49:])
	copy(market[:], key[57:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[89:])

	return
}

func EncodeOrderByTraderPrefix(trader solana.PublicKey) Prefix {
	key := make([]byte, 1+32)
	key[0] = PrefixOrderByTrader
	copy(key[1:], trader[:])
	return key
}

// 08:[trader]:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[rev_order_seq_num] => Pointer ot 06 key
func EncodeOrderByTraderMarket(trader, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+32+8+8+8+8)
	key[0] = PrefixOrderByTraderMarket
	copy(key[1:], trader[:])
	copy(key[33:], market[:])
	binary.BigEndian.PutUint64(key[65:], ^slotNum)
	binary.BigEndian.PutUint64(key[73:], ^trxIdx)
	binary.BigEndian.PutUint64(key[81:], ^instIdx)
	binary.BigEndian.PutUint64(key[89:], ^orderSeqNum)
	return key
}

func DecodeOrderByTraderMarket(key Key) (trader solana.PublicKey, market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
	if key[0] != PrefixOrderByTraderMarket {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixOrderByTraderMarket))
	}
	copy(trader[:], key[1:])
	copy(market[:], key[33:])
	slotNum = ^binary.BigEndian.Uint64(key[65:])
	trxIdx = ^binary.BigEndian.Uint64(key[73:])
	instIdx = ^binary.BigEndian.Uint64(key[81:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[89:])

	return
}

func EncodeOrderByTraderMarketPrefix(trader, market solana.PublicKey) Prefix {
	key := make([]byte, 1+32+32)
	key[0] = PrefixOrderByTraderMarket
	copy(key[1:], trader[:])
	copy(key[33:], market[:])
	return key
}

// 06:[market]:[rev_order_seq_num]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[trader] => Order
func EncodeOrderByMarket(trader, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+8+8+8+8+32)
	key[0] = PrefixOrderByMarket
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], ^orderSeqNum)
	binary.BigEndian.PutUint64(key[41:], ^slotNum)
	binary.BigEndian.PutUint64(key[49:], ^trxIdx)
	binary.BigEndian.PutUint64(key[57:], ^instIdx)
	copy(key[65:], trader[:])
	return key
}

func DecodeOrderByMarket(key Key) (trader solana.PublicKey, market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
	if key[0] != PrefixOrderByMarket {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixOrderByMarket))
	}
	copy(market[:], key[1:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[33:])
	slotNum = ^binary.BigEndian.Uint64(key[41:])
	trxIdx = ^binary.BigEndian.Uint64(key[49:])
	instIdx = ^binary.BigEndian.Uint64(key[57:])
	copy(trader[:], key[65:])
	return
}

func EncodeOrderByMarketPrefixWithOrder(market solana.PublicKey, orderSeqNum uint64) Prefix {
	key := make([]byte, 1+32+8)
	key[0] = PrefixOrderByMarket
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], ^orderSeqNum)
	return key
}

func EncodeOrderByMarketPrefix(market solana.PublicKey) Prefix {
	key := make([]byte, 1+32+8)
	key[0] = PrefixOrderByMarket
	copy(key[1:], market[:])
	return key
}

// 05:[trading_account] => [trader]
func EncodeTradingAccount(tradingAccount solana.PublicKey) Key {
	key := make([]byte, 1+32)

	key[0] = PrefixTradingAccount
	copy(key[1:], tradingAccount[:])

	return key
}

func DecodeTradingAccount(key Key) (tradingAccount solana.PublicKey) {
	if key[0] != PrefixTradingAccount {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixFillByTraderMarket))
	}

	key[0] = PrefixTradingAccount
	copy(tradingAccount[:], key[1:])
	return
}

func StartOfTradingAccount() Key { return []byte{PrefixTradingAccount} }

func EndOfTradingAccount() Key { return []byte{PrefixTradingAccount + 1} }

func EncodeCheckpoint() []byte {
	return []byte{PrefixCheckpoint, 0x6c, 0x61, 0x73, 0x74} // last in ascii
}

type Prefix []byte

func (p Prefix) String() string { return hex.EncodeToString(p) }

type Key []byte

func (k Key) String() string { return hex.EncodeToString(k) }

type KeyDecoder func(Key) (solana.PublicKey, solana.PublicKey, uint64, uint64, uint64, uint64)
