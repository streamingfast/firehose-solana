package server

import (
	"os"

	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var traceEnabled = os.Getenv("TRACE") != ""
var zlog = zap.NewNop()

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/graphql/server", &zlog)
}
