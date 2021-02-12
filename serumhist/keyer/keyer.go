package keyer

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/solana-go"
)

const (
	PrefixFill               = byte(0x01) // 01:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[rev_order_seq_num] 			=> FillData with trader
	PrefixFillByTrader       = byte(0x02) // 02:[trader]:[rev_slot_num]:[rev_trx_index]:[]:[market]:[rev_order_seq_num]							=> null
	PrefixFillByTraderMarket = byte(0x03) // 03:[trader]:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[rev_order_seq_num] 	=> null

	PrefixTradingAccount = byte(0x05) // 05:[trading_account] => [trader]

	// 06:[market]:[rev_order_seq_num]:<OrderEventTypeNew>:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index] => (new Order) => Order with trader
	// 06:[market]:[rev_order_seq_num]:<OrderEventTypeFill>:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index] => null
	// 06:[market]:[rev_order_seq_num]:<OrderEventTypeFill>:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index] => null
	// 06:[market]:[rev_order_seq_num]:<OrderEventTypeFill>:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]  => null
	// 06:[market]:[rev_order_seq_num]:<OrderEventTypeExecuted>:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index] => (executed) => null
	// 06:[market]:[rev_order_seq_num]:<OrderEventTypeCancel>:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index] => (cancelled) => null
	// 06:[market]:[rev_order_seq_num]:<OrderEventTypeClose>:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index] => (close (used to support serum V1 request queue & matching orders) => null
	PrefixOrder            = byte(0x06)
	OrderEventTypeNew      = byte(0x01)
	OrderEventTypeFill     = byte(0x02)
	OrderEventTypeExecuted = byte(0x03)
	OrderEventTypeCancel   = byte(0x04)
	OrderEventTypeClose    = byte(0x05)

	PrefixOrderByMarket       = byte(0x07) // 07:[market]:[rev_order_seq_num] => null
	PrefixOrderByTrader       = byte(0x08) // 08:[trader]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[market]:[rev_order_seq_num] => null
	PrefixOrderByTraderMarket = byte(0x08) // 08:[trader]:[:market]:[rev_order_seq_num] => null

	PrefixCheckpoint = byte(0x10)
)

// 01:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[rev_order_seq_num] 			=> FillData with trader
func EncodeFill(market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+8+8+8+8)
	key[0] = PrefixFill
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], ^slotNum)
	binary.BigEndian.PutUint64(key[41:], ^trxIdx)
	binary.BigEndian.PutUint64(key[49:], ^instIdx)
	binary.BigEndian.PutUint64(key[57:], ^orderSeqNum)
	return key
}

func DecodeFill(key Key) (market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
	if key[0] != PrefixFill {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixFill))
	}
	copy(market[:], key[1:])
	slotNum = ^binary.BigEndian.Uint64(key[33:])
	trxIdx = ^binary.BigEndian.Uint64(key[41:])
	instIdx = ^binary.BigEndian.Uint64(key[49:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[57:])
	return
}

// 02:[trader]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[market]:[rev_order_seq_num] 	=> null
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

func EncodeFillByMarketPrefix(market solana.PublicKey) Prefix {
	key := make([]byte, 1+32)
	key[0] = PrefixFillByTrader
	copy(key[1:], market[:])
	return key
}

// 03:[trader]:[market]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[rev_order_seq_num] 	=> null
func EncodeFillByTraderMarket(trader, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
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
func DecodeFillByTraderMarket(key Key) (trader solana.PublicKey, market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
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
func EncodeFillByTraderMarketPrefix(trader, market solana.PublicKey) Prefix {
	key := make([]byte, 1+32+32)
	key[0] = PrefixFillByTraderMarket
	copy(key[1:], trader[:])
	copy(key[33:], market[:])
	return key
}

func EncodeOrderNew(market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	return encodeOrder(OrderEventTypeNew, market, slotNum, trxIdx, instIdx, orderSeqNum)
}
func EncodeOrderFill(market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	return encodeOrder(OrderEventTypeFill, market, slotNum, trxIdx, instIdx, orderSeqNum)
}
func EncodeOrderCancel(market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	return encodeOrder(OrderEventTypeCancel, market, slotNum, trxIdx, instIdx, orderSeqNum)
}

// 06:[market]:[rev_order_seq_num]:<EVENT_BYTE>:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]
func encodeOrder(event byte, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+8+1+8+8+8)
	key[0] = PrefixOrder
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], ^orderSeqNum)
	key[41] = event
	binary.BigEndian.PutUint64(key[42:], ^slotNum)
	binary.BigEndian.PutUint64(key[50:], ^trxIdx)
	binary.BigEndian.PutUint64(key[58:], ^instIdx)
	return key
}

func DecodeOrder(key Key) (event byte, market solana.PublicKey, slotNum, trxIdx, instIdx, orderSeqNum uint64) {
	if key[0] != PrefixOrder {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixOrder))
	}
	copy(market[:], key[1:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[33:])
	event = key[41]
	slotNum = ^binary.BigEndian.Uint64(key[42:])
	trxIdx = ^binary.BigEndian.Uint64(key[50:])
	instIdx = ^binary.BigEndian.Uint64(key[58:])
	return
}

// 07:[market]:[rev_order_seq_num] => null
func EncodeOrderByMarket(market solana.PublicKey, orderSeqNum uint64) Key {
	key := make([]byte, 1+32+8)
	key[0] = PrefixOrderByMarket
	copy(key[1:], market[:])
	binary.BigEndian.PutUint64(key[33:], ^orderSeqNum)
	return key
}
func DecodeOrderByMarket(key Key) (market solana.PublicKey, orderSeqNum uint64) {
	if key[0] != PrefixOrderByMarket {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixOrderByMarket))
	}
	copy(market[:], key[1:])
	orderSeqNum = ^binary.BigEndian.Uint64(key[33:])
	return
}
func EncodeOrderByMarketPrefix(market solana.PublicKey) Prefix {
	key := make([]byte, 1+32+8)
	key[0] = PrefixOrderByMarket
	copy(key[1:], market[:])
	return key
}

// 08:[trader]:[rev_slot_num]:[rev_trx_index]:[rev_instruction_index]:[market]:[rev_order_seq_num] => null
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

// 08:[trader]:[:market]:[rev_order_seq_num] => null
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
