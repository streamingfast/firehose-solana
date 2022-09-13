package tools

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog, tracer = logging.PackageLogger("tools", "github.com/streamingfast/firehose-solana/tools", logging.LoggerDefaultLevel(zap.InfoLevel))
