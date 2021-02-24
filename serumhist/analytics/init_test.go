package analytics

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/test-go/testify/require"
	"gorm.io/driver/bigquery"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testStore(t *testing.T) *Store {
	conn := "bigquery://dfuse-development-tools/us/serum_test"
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second, // Slow SQL threshold
			LogLevel:      logger.Info, // Log level
			Colorful:      true,        // Disable color
		},
	)

	db, err := gorm.Open(bigquery.Open(conn), &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Silent),
		Logger: newLogger,
	})
	require.NoError(t, err)
	return NewStore(db)
}
