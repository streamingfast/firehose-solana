package transaction

import (
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var zlog = zap.NewNop()
var traceEnabled = logging.IsTraceEnabled("transaction", "github.com/dfuse-io/dfuse-solana/transaction")

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/transaction", &zlog)
}
