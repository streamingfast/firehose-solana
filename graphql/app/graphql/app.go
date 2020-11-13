package graphql

import (
	"github.com/dfuse-io/dfuse-solana/graphql/server"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	Name              string
	RPCURL            string
	HTTPListenAddress string
	RPCWSURL          string
}

type Modules struct {
}
type App struct {
	*shutter.Shutter
	config    *Config
	modules   *Modules
	ReadyFunc func()
	isReady   func() bool
}

func New(config *Config, modules *Modules) *App {
	return &App{
		Shutter:   shutter.New(),
		config:    config,
		modules:   modules,
		ReadyFunc: func() {},
	}
}

func (a *App) Run() error {
	zlog.Info("running graphql application", zap.Reflect("config", a.config))

	s := server.NewServer(
		a.config.HTTPListenAddress,
		a.config.RPCURL,
		a.config.RPCWSURL,
	)
	zlog.Info("serving ...")

	return s.Launch()
}
