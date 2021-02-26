package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/test-go/testify/require"
)

func TestLast24h(t *testing.T) {
	timeNow = func() time.Time {
		return parseTime(t, "2021-02-23 20:10:10")
	}

	date_range := last24h()
	assert.Equal(t, date_range.start, time.Date(2021, 02, 22, 20, 10, 10, 0, time.UTC))
	assert.Equal(t, date_range.stop, time.Date(2021, 02, 23, 20, 10, 10, 0, time.UTC))
}

func parseTime(t *testing.T, timeStr string) time.Time {
	t0, err := time.Parse("2006-01-02 15:04:05", timeStr)
	require.NoError(t, err)
	return t0
}

func TestStore_getSlotRange(t *testing.T) {
	store := testStore(t)

	date_range := &DateRange{
		start: parseTime(t, "2021-02-24 19:53:13"),
		stop:  parseTime(t, "2021-02-24 19:53:31"),
	}
	slotRange, err := store.getSlotRange(date_range)
	require.NoError(t, err)
	assert.Equal(t, &SlotNumRange{
		StartNum: 66693029,
		StopNum:  66693053,
	}, slotRange)
}
