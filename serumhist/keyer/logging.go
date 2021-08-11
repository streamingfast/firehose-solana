package keyer

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

var traceEnabled = logging.IsTraceEnabled("serumhist.keyer", "github.com/dfuse-io/dfuse-solana/serumhist/keyer")

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/serumhist/keyer", &zlog)
}
