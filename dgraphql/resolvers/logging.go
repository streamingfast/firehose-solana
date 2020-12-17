package resolvers

import (
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var zlog = zap.NewNop()

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/dgraphql/resolvers", &zlog)
}
