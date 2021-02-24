package analytics

import (
	"encoding/json"
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

func Last24h() *DateRange {
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
			dfuse-development-tools.serum_test.slot_timestamp 
		WHERE
			timestamp >= '2021-02-24 19:53:13' AND
			timestamp <= '2021-02-24 19:53:31'
		ORDER BY 
			timestamp ASC
		LIMIT 1`

	trx := s.db.Raw(query).Scan(&slot)
	if trx.Error != nil {
		return nil, fmt.Errorf("unable to retrieve slot range: %w", trx.Error)
	}
	if trx.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return &slot, nil
}

func (s *Store) scanSlotTimestamp() ([]*SlotTimestamp, error) {
	var outs []*SlotTimestamp
	query := `
		SELECT
			*
		FROM 
			dfuse-development-tools.serum_test.slot_timestamp
		ORDER BY 
			timestamp ASC
		`

	trx := s.db.Raw(query).Scan(&outs)
	if trx.Error != nil {
		return nil, fmt.Errorf("unable to retrieve slot range: %w", trx.Error)
	}
	fmt.Println(len(outs))
	for _, slotTimestamp := range outs {
		cnt, _ := json.Marshal(slotTimestamp)
		fmt.Println(string(cnt))
	}

	return outs, nil
}
