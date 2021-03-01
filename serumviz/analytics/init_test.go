package analytics

import (
	"testing"

	"github.com/test-go/testify/require"
	"gorm.io/driver/bigquery"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testStore(t *testing.T) *Store {
	conn := "bigquery://dfuse-development-tools/us/serum_test"
	//conn := "bigquery://dfuseio-global/us/serum"
	db, err := gorm.Open(bigquery.Open(conn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return NewStore(db)
}
