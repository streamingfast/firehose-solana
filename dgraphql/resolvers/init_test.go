package resolvers

import (
	"github.com/streamingfast/solana-go"
)

var pubKey = solana.MustPublicKeyFromBase58

type serumTestAnalyzable struct {
	volume24h float64
}
