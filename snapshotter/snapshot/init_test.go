package snapshot

import (
	_ "github.com/dfuse-io/kvdb/store/badger"
	"github.com/dfuse-io/logging"
)

func init() {
	logging.TestingOverride()
}
