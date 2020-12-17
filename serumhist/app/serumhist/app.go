package serumhist

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dfuse-io/dfuse-solana/serumhist"

	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"
	"github.com/dfuse-io/dmetrics"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	BlockStreamAddr string
	BatchSize       uint64
	StartBlock      uint64
	KvdbDsn         string
	HTTPListenAddr  string //  http listen address for /healthz endpoint
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

	kvdb, err := store.New(a.Config.KvdbDsn)
	if err != nil {
		zlog.Fatal("could not create kvstore", zap.Error(err))
	}
	kvdb = serumhist.NewRWCache(kvdb)

	dmetrics.Register(metrics.Metricset)

	loader := serumhist.NewLoader(a.Config.BlockStreamAddr, kvdb, a.Config.BatchSize)
	if err := loader.SetupLoader(); err != nil {
		return fmt.Errorf("unable to create solana loader: %w", err)
	}

	healthzSer, err := a.SetupHealthzServer(a.Config.HTTPListenAddr, loader)
	if err != nil {
		return fmt.Errorf("unable to setup health server: %w", err)
	}

	zlog.Info("starting webserver", zap.String("http_addr", a.Config.HTTPListenAddr))
	go healthzSer.ListenAndServe()

	a.OnTerminating(loader.Shutdown)
	loader.OnTerminated(a.Shutdown)

	go loader.Launch(context.Background(), a.Config.StartBlock)
	return nil
}

func (a *App) SetupHealthzServer(HTTPListenAddr string, loader *serumhist.Loader) (*http.Server, error) {
	healthzHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loader.Healthy() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		w.Write([]byte("ready\n"))
	})

	errorLogger, err := zap.NewStdLogAt(zlog, zap.ErrorLevel)
	if err != nil {
		return nil, fmt.Errorf("unable to create error logger: %w", err)
	}

	httpSrv := &http.Server{
		Addr:     HTTPListenAddr,
		Handler:  healthzHandler,
		ErrorLog: errorLogger,
	}
	return httpSrv, nil
}

func (a *App) IsReady() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	url := fmt.Sprintf("http://%s/healthz", a.Config.HTTPListenAddr)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		zlog.Warn("is ready request building error", zap.Error(err))
		return false
	}
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		zlog.Debug("is ready request execution error", zap.Error(err))
		return false
	}

	if res.StatusCode == 200 {
		return true
	}
	return false
}
