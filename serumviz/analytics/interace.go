package analytics

import "github.com/dfuse-io/solana-go"

type Analyzable interface {
	Get24hVolume() (float64, error)
	GetHourlyFillsVolume(*DateRange, *solana.PublicKey) ([]*FillVolume, error)
}
