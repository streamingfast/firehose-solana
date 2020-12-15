package solanadb_loader

import (
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/solanadb-loader/app/solanadb-loader", &zlog)
}
