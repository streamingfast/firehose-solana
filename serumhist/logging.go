package serumhist

import (
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

var traceEnabled = logging.IsTraceEnabled("serumhist", "github.com/dfuse-io/dfuse-solana/serumhist")

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/serumhist", &zlog)
}