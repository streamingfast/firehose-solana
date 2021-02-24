package analytics

import (
	"encoding/json"
	"fmt"

	"github.com/dfuse-io/solana-go"
)

type FillVolume struct {
	USDVolume         string `gorm:"column:usd_volume" json:"usd_volume"`
	SlotNum           uint32 `gorm:"column:slot_num" json:"slot_num"`
	TrxIdx            uint32 `gorm:"column:trx_idx" json:"trx_idx"`
	InstIdx           uint32 `gorm:"column:inst_idx" json:"inst_idx"`
	MarketAddress     string `gorm:"column:market_address" json:"market_address"`
	BaseTokenAddress  string `gorm:"column:base_address" json:"base_token_address"`
	QuoteTokenAddress string `gorm:"column:quote_address" json:"quote_token_address"`
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

func (s *Store) Get24hVolume() {

	var fills []FillVolume

	s.db.Raw("SELECT * FROM volume_fills ORDER BY slot_num ASC").Scan(&fills)
	for _, fill := range fills {
		cnt, _ := json.Marshal(fill)
		fmt.Println(string(cnt))
	}
}

func (s *Store) totalFillsVolume(date_range *DateRange) (float64, error) {

	type result struct {
		Total float64
	}

	var r result
	query := `
		SELECT
			sum(usd_volume) as total
		FROM 
			volume_fills 
		WHERE
			timestamp >= ? AND
			timestamp <= ?
		`

	trx := s.db.Raw(query, date_range.start, date_range.stop).Scan(&r)
	if trx.Error != nil {
		return 0.0, fmt.Errorf("unable to retrieve slot range: %w", trx.Error)
	}
	if trx.RowsAffected == 0 {
		return 0.0, ErrNotFound
	}

	return r.Total, nil
}
