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

type DateRange struct {
	start time.Time
	stop  time.Time
}

var timeNow func() time.Time

func last24h() *DateRange {
	t0 := timeNow()
	return &DateRange{
		start: t0.Add(-24 * time.Hour),
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

func (s *Store) getSlotRange(date_range *DateRange) (*SlotNumRange, error) {
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

	trx := s.db.Raw(query, date_range.start, date_range.stop).Scan(&slot)
	if trx.Error != nil {
		return nil, fmt.Errorf("unable to retrieve slot range: %w", trx.Error)
	}
	if trx.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return &slot, nil
}
