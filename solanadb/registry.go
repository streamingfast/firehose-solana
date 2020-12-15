package solanadb

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var registry = make(map[string]DriverFactory)

type DriverFactory func(dsn string) (DB, error)

// Register registers a storage backend DB
func Register(schemeName string, factory DriverFactory) {
	schemeName = strings.ToLower(schemeName)

	if _, ok := registry[schemeName]; ok {
		panic(errors.Errorf("%s is already registered", schemeName))
	}

	registry[schemeName] = factory
}

func IsRegistered(schemeName string) bool {
	_, isRegistered := registry[schemeName]
	return isRegistered
}

// New initializes a new DB
func New(dsn string, opts ...Option) (DB, error) {
	return newFromDSN(dsn, opts)
}

func newFromDSN(dsnStr string, opts []Option) (DB, error) {
	zlog.Debug("new serumdb from dsn string", zap.String("dsn_string", dsnStr))
	dsns, factory, err := splitDsn(dsnStr)
	if err != nil {
		return nil, fmt.Errorf("dsn is not valid: %w", err)
	}

	if len(dsns) > 1 {
		return nil, fmt.Errorf("only supporting a single DSN for now, got %d", len(dsns))
	}

	zlog.Debug("serumdb instance factory", zap.Strings("dsns", dsns))

	driver, err := factory(dsns[0])
	if err != nil {
		return nil, err
	}

	zlog.Debug("configuring serumdb instance with options",
		zap.Stringer("type", reflect.TypeOf(driver)),
		zap.Int("opts_count", len(opts)),
	)

	for _, opt := range opts {
		err := opt(driver)
		if err != nil {
			return nil, fmt.Errorf("unable to apply option: %w", err)
		}
	}
	return driver, err
}

func splitDsn(dsns string) (out []string, factory DriverFactory, err error) {
	driverType := ""
	for _, dsn := range strings.Split(dsns, " ") {
		parts := strings.Split(dsn, "://")
		if len(parts) < 2 {
			return nil, nil, fmt.Errorf("missing :// in DSN")
		}

		if driverType != "" && parts[0] != driverType {
			return nil, nil, fmt.Errorf("serumdb does not support splitting across musltiple driver types")
		}
		driverType = parts[0]

		factory = registry[driverType]
		if factory == nil {
			return nil, nil, fmt.Errorf("dsn: unregistered driver for scheme %q, have you '_ import'ed the package?", parts[0])
		}

		out = append(out, dsn)
	}
	return out, factory, nil
}
