package server

import (
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var traceEnabled = logging.IsTraceEnabled("dfusesol", "github.com/dfuse-io/dfuse-solana/graphql/server")
var zlog = zap.NewNop()

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/graphql/server", &zlog)
}
