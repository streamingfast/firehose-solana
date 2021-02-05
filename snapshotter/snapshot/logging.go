package snapshot

import (
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

var traceEnabled = logging.IsTraceEnabled("snapshotter", "github.com/dfuse-io/dfuse-solana/snapshotter/snapshot")

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/snapshotter/snapshot", &zlog)
}
