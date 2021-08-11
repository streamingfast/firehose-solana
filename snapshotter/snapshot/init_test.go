package snapshot

import (
	_ "github.com/streamingfast/kvdb/store/badger"
	"github.com/streamingfast/logging"
)

func init() {
	logging.TestingOverride()
}
