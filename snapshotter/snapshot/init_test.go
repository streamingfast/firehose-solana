package snapshot

import (
	_ "github.com/streamingfast/kvdb/store/badger"
	"github.com/dfuse-io/logging"
)

func init() {
	logging.TestingOverride()
}
