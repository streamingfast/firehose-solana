package resolvers

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog = zap.NewNop()

func init() {
	logging.Register("github.com/streamingfast/sf-solana/dgraphql/resolvers", &zlog)
}
