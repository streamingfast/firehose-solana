package pbserumhist

import (
	"testing"

	"github.com/test-go/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestFill_GetPrice(t *testing.T) {
	f := &Fill{
		OrderId: "0000000000000eedffffffffffa78933",
	}
	price, err := f.GetPrice()
	require.NoError(t, err)
	assert.Equal(t, uint64(3821), price)
}

func TestFill_GetSeqNum(t *testing.T) {
	f := &Fill{
		OrderId: "0000000000000eedffffffffffa78933",
		Side:    Side_BID,
	}
	price, err := f.GetSeqNum()
	require.NoError(t, err)
	assert.Equal(t, uint64(5797580), price)

	f = &Fill{
		OrderId: "0000000000000eed00000000005876cc",
		Side:    Side_ASK,
	}
	price, err = f.GetSeqNum()
	require.NoError(t, err)
	assert.Equal(t, uint64(5797580), price)
}
