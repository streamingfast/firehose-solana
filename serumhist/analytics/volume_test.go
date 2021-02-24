package analytics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Get24hVolume(t *testing.T) {
	store := testStore(t)

	date_range := &DateRange{
		start: parseTime(t, "2021-02-24 19:53:13"),
		stop:  parseTime(t, "2021-02-24 19:53:31"),
	}
	usd_volume, err := store.totalFillsVolume(date_range)
	require.NoError(t, err)
	assert.Equal(t, 130.727944, usd_volume)

}
