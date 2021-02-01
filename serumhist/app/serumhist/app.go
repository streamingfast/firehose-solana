package serumhist

import (
	"context"
	"errors"
	"fmt"

	"github.com/dfuse-io/dstore"

	"github.com/dfuse-io/dfuse-solana/serumhist/grpc"

	"github.com/dfuse-io/dfuse-solana/serumhist"

	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"
	"github.com/dfuse-io/dmetrics"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	BlockStreamAddr          string
	BlocksStoreURL           string
	FLushSlotInterval        uint64
	StartBlock               uint64
	KvdbDsn                  string
	EnableInjector           bool
	EnableServer             bool
	GRPCListenAddr           string
	HTTPListenAddr           string
	IgnoreCheckpointOnLaunch bool
}

type App struct {
	*shutter.Shutter
	Config *Config
}

func New(config *Config) *App {
	return &App{
		Shutter: shutter.New(),
		Config:  config,
	}
}

func (a *App) Run() error {
	zlog.Info("launching serumhist", zap.Reflect("config", a.Config))

	appCtx, cancel := context.WithCancel(context.Background())
	a.OnTerminating(func(err error) {
		cancel()
	})

	if err := a.Config.validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	kvdb, err := store.New(a.Config.KvdbDsn)
	if err != nil {
		zlog.Fatal("could not create kvstore", zap.Error(err))
	}

	if a.Config.EnableServer {
		server := grpc.New(a.Config.GRPCListenAddr, serumhist.NewManager(kvdb))
		a.OnTerminating(server.Terminate)
		server.OnTerminated(a.Shutdown)
		go server.Serve()
	}

	if a.Config.EnableInjector {
		dmetrics.Register(metrics.Metricset)

		blocksStore, err := dstore.NewDBinStore(a.Config.BlocksStoreURL)
		if err != nil {
			return fmt.Errorf("failed setting up blocks store: %w", err)
		}

		injector := serumhist.NewInjector(appCtx, a.Config.BlockStreamAddr, blocksStore, kvdb, a.Config.FLushSlotInterval)
		err = injector.SetupSource(a.Config.StartBlock, a.Config.IgnoreCheckpointOnLaunch)
		if err != nil {
			return fmt.Errorf("unable to setup serumhist injector source: %w", err)
		}

		zlog.Info("serum history injector setup complete")
		a.OnTerminating(func(err error) {
			injector.SetUnhealthy()
			injector.Shutdown(err)
		})
		injector.OnTerminated(a.Shutdown)

		injector.LaunchHealthz(a.Config.HTTPListenAddr)
		go func() {
			err := injector.Launch()
			if err != nil {
				zlog.Error("injector terminated with error")
				injector.Shutdown(err)
			} else {
				zlog.Info("injector terminated without error")
			}
		}()
	}

	return nil

}

func (c *Config) validate() error {
	if !c.EnableInjector && !c.EnableServer {
		return errors.New("both enable injection and enable server were disabled, this is invalid, at least one of them must be enabled, or both")
	}

	return nil
}
