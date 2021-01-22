package keyer

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/solana-go"
)

const (
	PrefixFillByTrader       = byte(0x01)
	PrefixFillByMarketTrader = byte(0x02)
	PrefixTradingAccount     = byte(0x05)
	PrefixCheckpoint         = byte(0x10)
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
	key[0] = PrefixFillByMarketTrader
	copy(key[1:], trader[:])
	copy(key[33:], market[:])
	binary.BigEndian.PutUint64(key[65:], ^slotNum)
	binary.BigEndian.PutUint64(key[73:], ^trxIdx)
	binary.BigEndian.PutUint64(key[81:], ^instIdx)
	binary.BigEndian.PutUint64(key[89:], ^orderSeqNum)
	return key
}

func DecodeFillByMarketTrader(key Key) (trader solana.PublicKey, market solana.PublicKey, slotNum uint64, trxIdx uint64, instIdx uint64, orderSeqNum uint64) {
	if key[0] != PrefixFillByMarketTrader {
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixFillByMarketTrader))
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
	key[0] = PrefixFillByMarketTrader
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
		panic(fmt.Sprintf("unable to decode key, expecting key prefix 0x%02x received: 0x%02x", key[0], PrefixFillByMarketTrader))
	}

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
