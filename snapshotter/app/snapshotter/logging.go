package snapshotter

import (
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

func init() {
	logging.Register("github.com/dfuse-io/dfuse-solana/serumhist/app/snapshotter", &zlog)
}
