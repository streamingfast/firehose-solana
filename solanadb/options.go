package solanadb

import "go.uber.org/zap"

type Option func(db DB) error

func WithLogger(logger *zap.Logger) Option {
	type loggeableStore interface {
		SetLogger(*zap.Logger) error
	}

	return func(db DB) error {
		if d, ok := db.(loggeableStore); ok {
			return d.SetLogger(logger)
		}
		return nil
	}
}
