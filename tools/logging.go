package tools

import (
	"os"

	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var traceEnabled = os.Getenv("TRACE") == "true"
var zlog = zap.NewNop()

func init() {
	logging.RegisterLogger("github.com/streamingfast/sf-solana/tools", zlog)
}
