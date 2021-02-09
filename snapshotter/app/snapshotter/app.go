package snapshotter

import (
	"github.com/dfuse-io/dfuse-solana/snapshotter/snapshot"
	"github.com/dfuse-io/shutter"
)

type Config struct {
	SourceBucket              string
	SourceSnapshopPrefix      string
	Workdir                   string
	DestinationSnapshotPrefix string
	DestinationBucket         string
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

	app.finder = snapshot.NewFinder(config.SourceBucket, config.SourceSnapshopPrefix, config.DestinationBucket, config.DestinationSnapshotPrefix, config.Workdir)
	app.finder.OnTerminating(func(err error) {
		app.Shutdown(err)
	})

	return app
}

func (a *App) Run() error {
	zlog.Info("Running")
	return a.finder.Launch()
}
