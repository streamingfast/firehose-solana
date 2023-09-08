package accountsresolver

import (
	"github.com/mr-tron/base58"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Keyer_ExtendTableLookup(t *testing.T) {
	var a1 = "2iMPmzAgkUWRjq1E5C4gAFA7bDKCBUrUbogGd8dau5XP"
	trxHashBytes, err := base58.Decode("VSod23zXfXD7RY9mPDuAJBkb674gZJ6n3CZUKT58Y4wCzFdcLLouCJkgNsG24Srkez7JK3mp6ozCiirojSbBG5u")
	require.NoError(t, err)

	key := Keys.extendTableLookup(accountFromBase58(t, a1), 1, trxHashBytes)
	expectedKey := []byte{tableAccountLookup}
	expectedKey = append(expectedKey, accountFromBase58(t, a1)...)
	expectedKey = append(expectedKey, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe}...)
	expectedKey = append(expectedKey, trxHashBytes...)
	require.Equal(t, expectedKey, key)
}

func Test_Keyer_Cusor(t *testing.T) {
	key := Keys.cursor("reader_1")
	expectedKey := []byte{tableCursor}
	expectedKey = append(expectedKey, []byte("reader_1")...)
	require.Equal(t, expectedKey, key)
}

func Test_Keyer_UnpackTableLookup(t *testing.T) {
	expectedAccount := "13Y2WX93BgJa7xhEQHokNkuVoFgk4p9vwAAT3aTkj87"
	expectedBlockNum := uint64(157564936)
	expectedTrxHash := "VSod23zXfXD7RY9mPDuAJBkb674gZJ6n3CZUKT58Y4wCzFdcLLouCJkgNsG24Srkez7JK3mp6ozCiirojSbBG5u"
	key := []byte{0, 0, 2, 221, 194, 243, 179, 183, 173, 114, 231, 92, 149, 174, 86, 70, 107, 79, 77, 133, 179, 2, 64, 248, 58, 81, 225, 250, 60, 184, 217, 59, 252, 255, 255, 255, 255, 246, 155, 191, 247, 24, 135, 160, 185, 200, 241, 239, 246, 95, 5, 218, 34, 45, 47, 87, 212, 109, 231, 185, 43, 190, 44, 64, 140, 192, 109, 59, 58, 213, 188, 210, 224, 94, 111, 208, 187, 34, 20, 205, 102, 155, 253, 129, 6, 146, 119, 140, 163, 187, 33, 35, 154, 95, 122, 98, 226, 246, 6, 133, 222, 231, 221, 21, 8}

	acc, blockNum, hashBytes := Keys.unpackTableLookup(key)
	require.Equal(t, expectedAccount, base58.Encode(acc))
	require.Equal(t, expectedBlockNum, blockNum)
	require.Equal(t, expectedTrxHash, base58.Encode(hashBytes))
}
