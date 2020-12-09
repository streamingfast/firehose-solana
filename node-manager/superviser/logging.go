package superviser

import (
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var zlog = zap.NewNop()
var traceEnabled = logging.IsTraceEnabled("superviser", "github.com/dfuse-io/dfuse-solana/node-manager/superviser")

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/node-manager/superviser", &zlog)
}
