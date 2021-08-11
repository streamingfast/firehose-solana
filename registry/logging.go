package registry

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog = zap.NewNop()
var traceEnabled = logging.IsTraceEnabled("token", "github.com/streamingfast/sf-solana/token")

func init() {
	logging.Register("github.com/streamingfast/sf-solana/token", &zlog)
}
