package nodemanager

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var traceEnabled = logging.IsTraceEnabled("serumhist", "github.com/streamingfast/sf-solana/nodemanager")
var zlog *zap.Logger

func init() {
	logging.Register("github.com/streamingfast/sf-solana/nodemanager", &zlog)
}

type stringArray []string

func (ss stringArray) MarshalLogArray(arr zapcore.ArrayEncoder) error {
	for _, element := range ss {
		arr.AppendString(element)
	}

	return nil
}
