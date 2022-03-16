package snapshot

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

var traceEnabled = logging.IsTraceEnabled("snapshotter", "github.com/streamingfast/sf-solana/snapshotter/snapshot")

func init() {
	logging.RegisterLogger("github.com/streamingfast/sf-solana/snapshotter/snapshot", zlog)
}
