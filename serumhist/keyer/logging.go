package keyer

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

var traceEnabled = logging.IsTraceEnabled("serumhist.keyer", "github.com/streamingfast/sf-solana/serumhist/keyer")

func init() {
	logging.Register("github.com/streamingfast/sf-solana/serumhist/keyer", &zlog)
}
