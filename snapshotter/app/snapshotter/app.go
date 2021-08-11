package snapshotter

import (
	"github.com/streamingfast/sf-solana/snapshotter/snapshot"
	"github.com/streamingfast/shutter"
)

type Config struct {
	SourceBucket               string
	SourceSnapshotsFolder      string
	Workdir                    string
	DestinationSnapshotsFolder string
	DestinationBucket          string
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

	app.finder = snapshot.NewFinder(config.SourceBucket, config.SourceSnapshotsFolder, config.DestinationBucket, config.DestinationSnapshotsFolder, config.Workdir)
	app.finder.OnTerminating(func(err error) {
		app.Shutdown(err)
	})

	return app
}

func (a *App) Run() error {
	zlog.Info("Running")
	return a.finder.Launch()
}
