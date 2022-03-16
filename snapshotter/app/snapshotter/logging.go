package snapshotter

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

func init() {
	logging.RegisterLogger("github.com/streamingfast/sf-solana/serumhist/app/snapshotter", zlog)
}
