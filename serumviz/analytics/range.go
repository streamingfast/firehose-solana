package analytics

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
)

func init() {
	timeNow = time.Now
}

type Granularity string

const (
	HourlyGranularity  Granularity = "hourly"
	DailyGranularity   Granularity = "daily"
	MonthlyGranularity Granularity = "monthly"
)

type DateRange struct {
	start time.Time
	stop  time.Time
}

var timeNow func() time.Time

func Last24Hours() DateRange {
	return newDateRangeWithDuration(24 * time.Hour)
}

func Last7Days() DateRange {
	return newDateRangeWithDuration(7 * 24 * time.Hour)
}

func Last30Days() DateRange {
	return newDateRangeWithDuration(30 * 14 * time.Hour)
}

func newDateRangeWithDuration(hours time.Duration) DateRange {
	t0 := timeNow()
	return DateRange{
		start: t0.Add(-1 * hours),
		stop:  t0,
	}
}

type SlotNumRange struct {
	StartNum uint32 `gorm:"column:start_num" json:"start_num"`
	StopNum  uint32 `gorm:"column:stop_num" json:"stop_num"`
}

type SlotTimestamp struct {
	SlotNum   int       `gorm:"column:slot_num" json:"slot_num"`
	Timestamp time.Time `gorm:"column:timestamp" json:"timestamp"`
}

func (s *Store) getSlotRange(dateRange *DateRange) (*SlotNumRange, error) {
	var slot SlotNumRange
	query := `
		SELECT
			FIRST_VALUE(slot_num) OVER (PARTITION BY 1 ORDER BY timestamp ASC) AS start_num,
			FIRST_VALUE(slot_num) OVER (PARTITION BY 1 ORDER BY timestamp DESC) AS stop_num
		FROM
			slot_timestamp
		WHERE
			timestamp >= ? AND
			timestamp <= ?
		ORDER BY
			timestamp ASC
		LIMIT 1`

	trx := s.db.Raw(query, dateRange.start, dateRange.stop).Scan(&slot)
	if trx.Error != nil {
		return nil, fmt.Errorf("unable to retrieve slot range: %w", trx.Error)
	}
	if trx.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return &slot, nil
}
