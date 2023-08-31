package solana_accounts_resolver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Keyer_ExtendTableLookup(t *testing.T) {
	var a1 = "2iMPmzAgkUWRjq1E5C4gAFA7bDKCBUrUbogGd8dau5XP"

	key := Keys.extendTableLookup(testAccountFromBase58(a1), 1)
	expectedKey := []byte{tableAccountLookup}
	expectedKey = append(expectedKey, testAccountFromBase58(a1)...)
	expectedKey = append(expectedKey, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe}...)
	require.Equal(t, expectedKey, key)
}

func Test_Keyer_Cusor(t *testing.T) {
	key := Keys.cursor("reader_1")
	expectedKey := []byte{tableCursor}
	expectedKey = append(expectedKey, []byte("reader_1")...)
	require.Equal(t, expectedKey, key)
}
