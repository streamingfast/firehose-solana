package analytics

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStore_TotalVolume(t *testing.T) {
	t.Skip("Do not run long running process")
	store := testStore(t)
	tests := []struct {
		name        string
		dateRange   DateRange
		expectValue float64
	}{
		{
			name:        "Last 24 Hours",
			dateRange:   Last24Hours(),
			expectValue: 0,
		},
		{
			name:        "Last 7 Days",
			dateRange:   Last7Days(),
			expectValue: 0,
		},
		{
			name:        "Last 30 Days",
			dateRange:   Last30Days(),
			expectValue: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := store.TotalVolume(test.dateRange)
			require.NoError(t, err)
			fmt.Printf("%s: %f\n", test.name, value)
		})
	}
}

func TestStore_FillsVolume(t *testing.T) {
	t.Skip("Do not run long running process")
	store := testStore(t)
	tests := []struct {
		name        string
		dateRange   DateRange
		granularity Granularity
		expectVaue  float64
	}{
		{
			name:        "Last 24 Hours",
			dateRange:   Last24Hours(),
			granularity: HourlyGranularity,
			expectVaue:  0,
		},
		{
			name:        "Last 7 Days",
			dateRange:   Last7Days(),
			granularity: DailyGranularity,
			expectVaue:  0,
		},
		{
			name:        "Last 30 Days",
			dateRange:   Last30Days(),
			granularity: MonthlyGranularity,
			expectVaue:  0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fills, err := store.FillsVolume(&test.dateRange, &test.granularity, nil)
			require.NoError(t, err)
			fmt.Printf("Fills granularity %s\n", test.granularity)
			for _, fill := range fills {
				cnt, _ := json.Marshal(fill)
				fmt.Println(string(cnt))
			}

		})
	}
}
