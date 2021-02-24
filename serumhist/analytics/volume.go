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
