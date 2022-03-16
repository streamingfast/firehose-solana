package nodemanager

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap/zapcore"
)

var zlog, tracer = logging.PackageLogger("nodemanager", "github.com/streamingfast/sf-solana/nodemanager")

type stringArray []string

func (ss stringArray) MarshalLogArray(arr zapcore.ArrayEncoder) error {
	for _, element := range ss {
		arr.AppendString(element)
	}

	return nil
}
