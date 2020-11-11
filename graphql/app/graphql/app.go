package graphql

import (
	"github.com/dfuse-io/dfuse-solana/graphql"
	"github.com/dfuse-io/dfuse-solana/graphql/server"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	Name              string
	RPCEndpoint       string
	HTTPListenAddress string
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

	manager := graphql.NewManager()

	s := server.NewServer(
		a.config.HTTPListenAddress,
		a.config.RPCEndpoint,
		manager,
	)
	zlog.Info("serving ...")

	return s.Launch()
}
