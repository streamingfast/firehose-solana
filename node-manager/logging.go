package nodemanager

import "go.uber.org/zap/zapcore"

type stringArray []string

func (ss stringArray) MarshalLogArray(arr zapcore.ArrayEncoder) error {
	for _, element := range ss {
		arr.AppendString(element)
	}

	return nil
}
