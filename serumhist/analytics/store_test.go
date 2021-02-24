package analytics

import (
	"testing"

	"github.com/test-go/testify/require"
	"gorm.io/driver/bigquery"
	"gorm.io/gorm"
)

func TestNewStore(t *testing.T) {
	conn := "bigquery://dfuse-development-tools/us/serum"
	gorm.Open(bigquery.Open(conn), &gorm.Config{})
	db, err := gorm.Open(bigquery.Open(conn), &gorm.Config{})
	require.NoError(t, err)
	store := NewStore(db)
	store.Get24hVolume()
}
