package resolvers

import "github.com/dfuse-io/solana-go"

var pubKey = solana.MustPublicKeyFromBase58

type serumTestAnalyzable struct {
	volume24h float64
}

func (s *serumTestAnalyzable) Get24hVolume() (float64, error) {
	return s.volume24h, nil
}
