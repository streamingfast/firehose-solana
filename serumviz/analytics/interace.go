package analytics

import "github.com/streamingfast/solana-go"

type Analyzable interface {
	TotalVolume(DateRange) (float64, error)
	FillsVolume(*DateRange, *Granularity, *solana.PublicKey) ([]*FillVolume, error)
}
