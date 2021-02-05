package snapshotter

import (
	"github.com/dfuse-io/dfuse-solana/snapshotter/snapshot"
	"github.com/dfuse-io/shutter"
)

type Config struct {
	Bucket string
	Prefix string
}

type App struct {
	*shutter.Shutter
	config *Config
	finder *snapshot.Finder
}

func New(config *Config) *App {
	app := &App{
		Shutter: shutter.New(),
		config:  config,
	}

	app.finder = snapshot.NewFinder(config.Bucket, config.Prefix)
	app.finder.OnTerminating(func(err error) {
		app.Shutdown(err)
	})

	return app
}

func (a *App) Run() error {
	zlog.Info("Running")
	return a.finder.Launch()
}
