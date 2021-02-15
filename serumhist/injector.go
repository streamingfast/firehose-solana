package serumhist

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	"github.com/dfuse-io/bstream/firehose"
	"github.com/dfuse-io/bstream/forkable"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	serumhistdb "github.com/dfuse-io/dfuse-solana/serumhist/db"
	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"
	"github.com/dfuse-io/dfuse-solana/serumhist/reader"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/kvdb/store"
	pbhealth "github.com/dfuse-io/pbgo/grpc/health/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Injector struct {
	*shutter.Shutter
	ctx context.Context
	//db                      serumhistdb.DB
	flushSlotInterval uint64
	lastTickBlock     uint64
	lastTickTime      time.Time
	blockStore        dstore.Store
	blockstreamAddr   string
	healthy           bool

	source                  *firehose.Firehose
	slotMetrics             slotMetrics
	preprocessorThreadCount int

	handler            bstream.Handler
	checkpointResolver CheckpointResolver

	manager *OrderManager
	//reader                      *reader.Reader
	grpcAddr                    string
	server                      *dgrpc.Server
	parallelDownloadThreadCount int
}

// TODO don't depend on both....
func NewInjector(
	ctx context.Context,
	blockstreamAddr string,
	blockStore dstore.Store,
	db serumhistdb.DB,
	kvdb store.KVStore,
	flushSlotInterval uint64,
	preprocessorThreadCount int,
	parallelDownloadThreadCount int,
	grpcAddr string,
) *Injector {
	return &Injector{
		ctx:               ctx,
		blockstreamAddr:   blockstreamAddr,
		blockStore:        blockStore,
		Shutter:           shutter.New(),
		flushSlotInterval: flushSlotInterval,
		db:                db,
		cache:             kvloader.newTradingAccountCache(kvdb),
		slotMetrics: slotMetrics{
			startTime: time.Now(),
		},
		preprocessorThreadCount:     preprocessorThreadCount,
		parallelDownloadThreadCount: parallelDownloadThreadCount,
		grpcAddr:                    grpcAddr,
		manager:                     newOrderManager(),
		reader:                      reader.New(kvdb),
	}
}

func (i *Injector) SetupSource(startBlockNum uint64, ignoreCheckpointOnLaunch bool) error {
	zlog.Info("setting up serhumhist source",
		zap.Uint64("start_block_num", startBlockNum),
	)

	checkpoint, err := i.resolveCheckpoint(i.ctx, startBlockNum, ignoreCheckpointOnLaunch)
	if err != nil {
		return fmt.Errorf("unable to resolve shard checkpoint: %w", err)
	}

	zlog.Info("serumhist resolved start block",
		zap.Uint64("start_block_num", checkpoint.LastWrittenSlotNum),
		zap.String("start_block_id", checkpoint.LastWrittenSlotId),
	)

	options := []firehose.Option{
		firehose.WithLogger(zlog),
		firehose.WithForkableSteps(forkable.StepNew | forkable.StepIrreversible),
	}

	if i.blockstreamAddr != "" {
		liveStreamFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			return blockstream.NewSource(
				i.ctx,
				i.blockstreamAddr,
				200,
				subHandler,
				blockstream.WithParallelPreproc(i.preprocessSlot, i.preprocessorThreadCount),
			)
		})
		options = append(options, firehose.WithLiveSource(liveStreamFactory, false))
	}

	fileSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		fs := bstream.NewFileSource(
			i.blockStore,
			startBlockNum,
			i.parallelDownloadThreadCount,
			i.preprocessSlot,
			h,
			bstream.FileSourceWithConcurrentPreprocess(i.preprocessorThreadCount),
		)
		return fs
	})

	fhose := firehose.New(
		fileSourceFactory,
		int64(checkpoint.LastWrittenSlotNum),
		i,
		options...,
	)
	i.source = fhose
	return nil
}

func (i *Injector) Launch() error {
	zlog.Info("launching serumhist injector")

	if i.grpcAddr != "" {
		server := dgrpc.NewServer2(dgrpc.WithLogger(zlog))
		server.RegisterService(func(gs *grpc.Server) {
			pbserumhist.RegisterSerumOrderTrackerServer(gs, i)
			pbhealth.RegisterHealthServer(gs, i)
		})

		zlog.Info("listening for serum history",
			zap.String("addr", i.grpcAddr),
		)

		i.OnTerminating(func(err error) {
			server.Shutdown(30 * time.Second)
		})

		go server.Launch(i.grpcAddr)
	}

	err := i.source.Run(i.ctx)
	if err != nil {
		if errors.Is(err, firehose.ErrStopBlockReached) {
			zlog.Info("firehose stream of blocks reached end block")
			return nil
		}

		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("firehose context canceled")
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("firehose context deadline exceeded")
		}

		var e *firehose.ErrInvalidArg
		if errors.As(err, &e) {
			return fmt.Errorf("firehose invalid args: %s", e.Error())
		}

		return fmt.Errorf("firehose unexpected d error: %w", err)
	}
	return nil
}

func (i *Injector) doFlush(slotNum uint64, reason string) error {
	zlog.Debug("flushing block",
		zap.Uint64("slot_num", slotNum),
		zap.String("reason", reason),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	err := i.db.Flush(ctx)
	if err != nil {
		return fmt.Errorf("db flush: %w", err)
	}
	return nil
}

func (i *Injector) flushIfNeeded(slotNum uint64, slotTime time.Time) error {
	batchSizeReached := slotNum%i.flushSlotInterval == 0
	closeToHeadBlockTime := time.Since(slotTime) < 25*time.Second

	if batchSizeReached || closeToHeadBlockTime {
		reason := "needed"
		if batchSizeReached {
			reason += ", batch size reached"
		}

		if closeToHeadBlockTime {
			reason += ", close to head block"
		}

		err := i.doFlush(slotNum, reason)
		if err != nil {
			return err
		}
		metrics.HeadBlockNumber.SetUint64(slotNum)
	}

	return nil
}
