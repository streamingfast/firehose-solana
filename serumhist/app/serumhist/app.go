package serumhist

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/streamingfast/dmetrics"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/kvdb/store"
	"github.com/streamingfast/sf-solana/serumhist"
	"github.com/streamingfast/sf-solana/serumhist/grpc"
	kvloader "github.com/streamingfast/sf-solana/serumhist/kvloader"
	"github.com/streamingfast/sf-solana/serumhist/metrics"
	"github.com/streamingfast/sf-solana/serumhist/reader"
	bqloader "github.com/streamingfast/sf-solana/serumviz/bqloader"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
)

type Config struct {
	BlockStreamAddr           string
	BlocksStoreURL            string
	PreprocessorThreadCount   int
	MergeFileParallelDownload int

	IgnoreCheckpointOnLaunch bool
	FlushSlotInterval        uint64
	StartBlock               uint64

	EnableServer   bool
	EnableInjector bool
	GRPCListenAddr string
	HTTPListenAddr string

	EnableBigQueryInjector  bool
	KvdbDsn                 string
	BigQueryStoreURL        string
	BigQueryProject         string
	BigQueryDataset         string
	BigQueryScratchSpaceDir string

	TokensFileURL string
	MarketFileURL string
}

type App struct {
	*shutter.Shutter
	Config *Config
}

func New(config *Config) *App {
	//fail safe
	if config.PreprocessorThreadCount == 0 {
		config.PreprocessorThreadCount = 1
	}
	if config.MergeFileParallelDownload == 0 {
		config.MergeFileParallelDownload = 1
	}

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

	if a.Config.EnableServer {
		kvdb, err := store.New(a.Config.KvdbDsn)
		if err != nil {
			return fmt.Errorf("unable to create kvdb store: %w", err)
		}

		server := grpc.New(a.Config.GRPCListenAddr, reader.New(kvdb))
		server.OnTerminated(a.Shutdown)
		server.Serve()
	}

	if a.Config.EnableInjector {
		dmetrics.Register(metrics.Metricset)

		blocksStore, err := dstore.NewDBinStore(a.Config.BlocksStoreURL)
		if err != nil {
			return fmt.Errorf("failed setting up blocks store: %w", err)
		}

		handler, checkpointResolver, err := a.getHandler(appCtx)
		if err != nil {
			return fmt.Errorf("unable to create serumhist handler: %w", err)
		}

		injector := serumhist.NewInjector(appCtx, handler, checkpointResolver, a.Config.BlockStreamAddr, blocksStore, a.Config.PreprocessorThreadCount, a.Config.MergeFileParallelDownload)
		err = injector.SetupSource(a.Config.StartBlock, a.Config.IgnoreCheckpointOnLaunch)
		if err != nil {
			return fmt.Errorf("unable to setup serumhist injector source: %w", err)
		}

		zlog.Info("serum history injector setup complete")
		a.OnTerminating(func(err error) {
			injector.SetUnhealthy()
			injector.Shutdown(err)
		})

		injector.OnTerminated(func(err error) {
			handler.Shutdown(err)
			a.Shutdown(err)
		})

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

func (a *App) getHandler(ctx context.Context) (serumhist.Handler, serumhist.CheckpointResolver, error) {
	if a.Config.EnableBigQueryInjector {
		zlog.Info("setting up big query loader",
			zap.String("bigquery_project_id", a.Config.BigQueryProject),
			zap.String("bigquery_dataset_id", a.Config.BigQueryDataset),
		)
		bqClient, err := bigquery.NewClient(ctx, a.Config.BigQueryProject)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating bigquery client: %w", err)
		}

		dataset := bqClient.Dataset(a.Config.BigQueryDataset)

		// we do not want the avro files to be compress, or else bigquery would not be able to consume them!
		store, err := dstore.NewStore(a.Config.BigQueryStoreURL, "avro", "", false)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating bigquery dstore: %w", err)
		}

		loader := bqloader.New(ctx, a.Config.StartBlock, a.Config.BigQueryStoreURL, store, dataset, bqClient)

		err = loader.Init(ctx, a.Config.BigQueryScratchSpaceDir)
		if err != nil {
			return nil, nil, fmt.Errorf("error initializing event handlers: %w", err)
		}

		err = loader.PrimeTradeCache(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("error loading priming trading account cache from bigquery: %w", err)
		}

		return loader, loader.GetCheckpoint, nil
	}

	zlog.Info("setting up big kvdb loader",
		zap.String("dsn", a.Config.KvdbDsn),
	)
	kvdb, err := store.New(a.Config.KvdbDsn)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating kvdb store: %w", err)
	}
	loader := kvloader.NewLoader(ctx, kvdb, a.Config.FlushSlotInterval)
	loader.PrimeTradeCache(ctx)

	return loader, loader.GetCheckpoint, nil
}

func (c *Config) validate() error {
	if !c.EnableInjector && !c.EnableServer {
		return errors.New("both enable injection and enable server were disabled, this is invalid, at least one of them must be enabled, or both")
	}
	return nil
}
