package firehose

import (
	"context"
	"fmt"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	blockstreamv2 "github.com/dfuse-io/bstream/blockstream/v2"
	"github.com/dfuse-io/bstream/hub"
	_ "github.com/dfuse-io/dfuse-solana/codec"
	"github.com/dfuse-io/dgraphql/metrics"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dmetrics"
	"github.com/dfuse-io/dstore"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	BlocksStoreURL          string
	UpstreamBlockStreamAddr string
	GRPCListenAddr          string
}

type Modules struct {
	Tracker *bstream.Tracker
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
	dmetrics.Register(metrics.MetricSet)
	zlog.Info("running block stream", zap.Reflect("config", a.config))
	blocksStore, err := dstore.NewDBinStore(a.config.BlocksStoreURL)
	if err != nil {
		return fmt.Errorf("failed setting up blocks store: %w", err)
	}

	ctx := context.Background()
	var start uint64
	withLive := a.config.UpstreamBlockStreamAddr != ""
	if withLive {
		zlog.Info("starting with support for live blocks")
		for retries := 0; ; retries++ {
			lib, err := a.modules.Tracker.Get(ctx, bstream.BlockStreamLIBTarget)
			if err != nil {
				if retries%5 == 4 {
					zlog.Warn("cannot get lib num from blockstream, retrying", zap.Int("retries", retries), zap.Error(err))
				}
				time.Sleep(time.Second)
				continue
			}
			start = lib.Num()
			break
		}

	}

	liveSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		return blockstream.NewSource(
			context.Background(),
			a.config.UpstreamBlockStreamAddr,
			100,
			bstream.HandlerFunc(func(blk *bstream.Block, obj interface{}) error {
				metrics.HeadTimeDrift.SetBlockTime(blk.Time())
				return h.ProcessBlock(blk, obj)
			}),
			blockstream.WithRequester("blockstream"),
		)
	})

	fileSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		zlog.Info("creating file source", zap.Uint64("start_block_num", startBlockNum))
		src := bstream.NewFileSource(blocksStore, startBlockNum, 1, nil, h)
		return src
	})

	zlog.Info("setting up subscription hub")

	buffer := bstream.NewBuffer("hub-buffer", zlog.Named("hub"))
	tailManager := bstream.NewSimpleTailManager(buffer, 350)
	go tailManager.Launch()
	subscriptionHub, err := hub.NewSubscriptionHub(
		start,
		buffer,
		tailManager.TailLock,
		fileSourceFactory,
		liveSourceFactory,
		hub.Withlogger(zlog),
		hub.WithRealtimeTolerance(1*time.Minute),
		hub.WithoutMemoization(), // This should be tweakable on the Hub, by the bstreamv2.Server
	)
	if err != nil {
		return fmt.Errorf("setting up subscription hub: %w", err)
	}

	bsv2Tracker := a.modules.Tracker.Clone()

	zlog.Info("setting up blockstream V2 server")
	s := blockstreamv2.NewServer(bsv2Tracker, blocksStore, a.config.GRPCListenAddr, subscriptionHub)
	// s.SetPreprocFactory(func(req *pbbstream.BlocksRequestV2) (bstream.PreprocessFunc, error) {
	// 	filter, err := filtering.NewBlockFilter([]string{req.IncludeFilterExpr}, []string{req.ExcludeFilterExpr}, nil)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("parsing: %w", err)
	// 	}
	// 	preproc := &filtering.FilteringPreprocessor{Filter: filter}
	// 	return preproc.PreprocessBlock, nil
	// })

	a.isReady = s.IsReady
	go func() {
		subscriptionHub.Launch()
		if withLive {
			subscriptionHub.WaitReady()
		}
		zlog.Info("blockstream is now ready")
		s.SetReady()
		a.ReadyFunc()
	}()

	go func() {
		grpcSrv := dgrpc.NewServer(dgrpc.WithLogger(zlog))
		pbbstream.RegisterBlockStreamV2Server(grpcSrv, s)
		httpSrv := dgrpc.SimpleHTTPServer(grpcSrv, a.config.GRPCListenAddr, dgrpc.SimpleHealthCheck(a.IsTerminating))
		if err := dgrpc.ListenAndServe(httpSrv); err != nil {
			a.Shutdown(err)
		}
	}()

	return nil
}
