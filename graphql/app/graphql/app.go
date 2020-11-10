package graphql

import (
	"github.com/dfuse-io/dfuse-solana/graphql"
	"github.com/dfuse-io/shutter"
)

type Config struct {
	Name string
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
	server := graphql.NewServer(":8080", "http://api.mainnet-beta.solana.com:80/rpc", "ws://api.mainnet-beta.solana.com:80/rpc")
	zlog.Info("serving ...")
	return server.Launch()
}
