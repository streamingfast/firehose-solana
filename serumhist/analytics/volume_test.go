package analytics

import (
	"testing"
)

func TestStore_Get24hVolume(t *testing.T) {
	//conn := "bigquery://dfuse-development-tools/us/serum"
	//gorm.Open(bigquery.Open(conn), &gorm.Config{})
	//
	//newLogger := logger.New(
	//	log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
	//	logger.Config{
	//		SlowThreshold: time.Second, // Slow SQL threshold
	//		LogLevel:      logger.Info, // Log level
	//		Colorful:      true,        // Disable color
	//	},
	//)
	//
	//db, err := gorm.Open(bigquery.Open(conn), &gorm.Config{
	//	Logger: newLogger,
	//})
	//require.NoError(t, err)
	//store := NewStore(db)
	////store.Get24hVolume()
	//
	//date_range := &DateRange{
	//	start: parseTime(t, "2021-02-17 23:36:20"),
	//	stop:  parseTime(t, "2021-02-17 23:57:11"),
	//}
	////temp := store.getSlotRange(date_range)
	//
	//fmt.Println(temp)
}
