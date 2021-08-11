package grpc

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog *zap.Logger

var traceEnabled = logging.IsTraceEnabled("serumhist", "github.com/streamingfast/sf-solana/serumhist/grpc")

func init() {
	logging.Register("github.com/streamingfast/sf-solana/serumhist/grpc", &zlog)
}
