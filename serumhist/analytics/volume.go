package analytics

import (
	"fmt"
	"time"

	"github.com/dfuse-io/solana-go"
)

type FillVolume struct {
	USDVolume         string    `gorm:"column:usd_volume" json:"usd_volume"`
	Timestamp         time.Time `gorm:"column:timestamp" json:"timestamp"`
	SlotNum           uint32    `gorm:"column:slot_num" json:"slot_num"`
	TrxIdx            uint32    `gorm:"column:trx_idx" json:"trx_idx"`
	InstIdx           uint32    `gorm:"column:inst_idx" json:"inst_idx"`
	MarketAddress     string    `gorm:"column:market_address" json:"market_address"`
	BaseTokenAddress  string    `gorm:"column:base_address" json:"base_token_address"`
	QuoteTokenAddress string    `gorm:"column:quote_address" json:"quote_token_address"`
}

func (fv *FillVolume) Market() solana.PublicKey {
	return solana.MustPublicKeyFromBase58(fv.MarketAddress)
}

func (fv *FillVolume) BaseToken() solana.PublicKey {
	return solana.MustPublicKeyFromBase58(fv.BaseTokenAddress)
}

func (fv *FillVolume) QuoteToken() solana.PublicKey {
	return solana.MustPublicKeyFromBase58(fv.QuoteTokenAddress)
}

func (s *Store) Get24hVolume() (float64, error) {
	return s.totalFillsVolume(last24h())
}

func (s *Store) GetHourlyFillsVolume(date_range *DateRange, market *solana.PublicKey) ([]*FillVolume, error) {
	var out []*FillVolume

	query := s.db.Table("volume_fills").
		Select([]string{
			"sum(usd_volume) as usd_volume",
			"market_address",
			"TIMESTAMP(FORMAT_TIMESTAMP(\"%F %H:00:00\", timestamp)) as timestamp",
		})

	if date_range != nil {
		query = query.Where("timestamp >= ?", date_range.start).
			Where("timestamp <= ?", date_range.stop)
	}

	if market != nil {
		query = query.Where("market_address = ?", market.String())
	}

	query = query.Group("timestamp").Group("market_address")
	trx := query.Scan(&out)
	if trx.Error != nil {
		return nil, fmt.Errorf("unable to retrieve fils: %w", trx.Error)
	}
	return out, nil
}

func (s *Store) totalFillsVolume(date_range DateRange) (float64, error) {
	type result struct {
		Total float64
	}
	var r result

	trx := s.db.Table("volume_fills").
		Select("sum(usd_volume) as total").
		Where("timestamp >= ?", date_range.start).
		Where("timestamp <= ?", date_range.stop).
		Scan(&r)
	if trx.Error != nil {
		return 0.0, fmt.Errorf("unable to retrieve total fill: %w", trx.Error)
	}
	if trx.RowsAffected == 0 {
		return 0.0, ErrNotFound
	}

	return r.Total, nil
}
