package keyer

import (
	"fmt"
	"testing"

	"github.com/dfuse-io/solana-go"
	"github.com/stretchr/testify/assert"
)

func TestEncodeFillByTrader(t *testing.T) {
	traderkey := solana.PublicKey{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x22, 0x33,
	}

	marketkey := solana.PublicKey{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc,
	}

	key := EncodeFillByTrader(traderkey, marketkey, 2, 5, 2, 3)
	assert.Equal(t, Key([]byte{
		0x01,

		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x22, 0x33,

		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,

		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfa,

		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,

		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc,

		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfc,
	}), key)
}

func TestDecodeFillByTrader(t *testing.T) {
	trader, market, slotNum, trxIndx, InstIdx, orderSeqNum := DecodeFillByTrader([]byte{
		0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x22, 0x33,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfa,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfc,
	})
	assert.Equal(t, solana.PublicKey{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x22, 0x33}, trader)

	assert.Equal(t, solana.PublicKey{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc,
	}, market)
	assert.Equal(t, uint64(2), slotNum)
	assert.Equal(t, uint64(5), trxIndx)
	assert.Equal(t, uint64(2), InstIdx)
	assert.Equal(t, uint64(3), orderSeqNum)

}

func TestEncodeFillByMarketTrader(t *testing.T) {
	traderkey := solana.PublicKey{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x22, 0x33,
	}

	marketkey := solana.PublicKey{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc,
	}

	key := EncodeFillByMarketTrader(traderkey, marketkey, 2, 5, 2, 3)
	assert.Equal(t, Key([]byte{
		0x02,

		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x22, 0x33,

		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc,

		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,

		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfa,

		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,

		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfc,
	}), key)
}

func TestDecodeFillByMarketTrader(t *testing.T) {
	trader, market, slotNum, trxIdx, instIdx, orderSeqNum := DecodeFillByMarketTrader([]byte{
		0x02,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x22, 0x33,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfa,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfc,
	})
	assert.Equal(t, solana.PublicKey{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x11, 0x22, 0x33}, trader)

	assert.Equal(t, solana.PublicKey{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc,
	}, market)
	assert.Equal(t, slotNum, uint64(2))
	assert.Equal(t, trxIdx, uint64(5))
	assert.Equal(t, instIdx, uint64(2))
	assert.Equal(t, orderSeqNum, uint64(3))
}

func TestEncodeTradingAccount(t *testing.T) {
	tradingAccount := solana.PublicKey{
		0xaa, 0xbb, 0xcc, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	key := EncodeTradingAccount(tradingAccount)
	assert.Equal(t, Key([]byte{
		0x05,
		0xaa, 0xbb, 0xcc, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}), key)
}

func TestDecodeTradingAccount(t *testing.T) {
	tradingAccount := DecodeTradingAccount([]byte{
		0x05,
		0xaa, 0xbb, 0xcc, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	})

	assert.Equal(t, solana.PublicKey{
		0xaa, 0xbb, 0xcc, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}, tradingAccount)

}

func TestEncodeDecodeFillByTrader(t *testing.T) {
	k := EncodeFillByTrader(solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111111"), solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111112"), 123, 456, 789, 101112)
	trader, market, slotNum, trxIdx, instIdx, orderSeqNum := DecodeFillByTrader(k)

	fmt.Println("key", k)

	assert.Equal(t, solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111111"), trader)
	assert.Equal(t, solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111112"), market)
	assert.Equal(t, uint64(123), slotNum)
	assert.Equal(t, uint64(456), trxIdx)
	assert.Equal(t, uint64(789), instIdx)
	assert.Equal(t, uint64(101112), orderSeqNum)
}
