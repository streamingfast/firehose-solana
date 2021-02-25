package analytics

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dfuse-io/solana-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Get24hVolume(t *testing.T) {
	store := testStore(t)
	date_range := &DateRange{
		start: parseTime(t, "2021-02-24 19:53:13"),
		stop:  parseTime(t, "2021-02-24 19:53:31"),
	}
	usd_volume, err := store.totalFillsVolume(*date_range)
	require.NoError(t, err)
	assert.Equal(t, 130.727944, usd_volume)
}

func TestStore_GetHourlyFillsVolume(t *testing.T) {
	t.Skip("long running process")
	store := testStore(t)
	fills, err := store.GetHourlyFillsVolume(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 31, len(fills))

	for _, fill := range fills {
		cnt, _ := json.Marshal(fill)
		fmt.Println(string(cnt))
	}

	marketAddress := solana.MustPublicKeyFromBase58("hBswhpNyz4m5nt4KwtCA7jYXvh7VmyZ4TuuPmpaKQb1")
	fills, err = store.GetHourlyFillsVolume(nil, &marketAddress)
	require.NoError(t, err)
	assert.Equal(t, 2, len(fills))
}
