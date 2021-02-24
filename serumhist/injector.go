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
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Handler interface {
	bstream.Handler
	GetCheckpoint(ctx context.Context) (*pbserumhist.Checkpoint, error)
	Healthy() bool
	Close() error
}

type Injector struct {
	*shutter.Shutter
	ctx                         context.Context
	blockStore                  dstore.Store
	blockstreamAddr             string
	healthy                     bool
	source                      *firehose.Firehose
	slotMetrics                 slotMetrics
	preprocessorThreadCount     int
	parallelDownloadThreadCount int
	handler                     Handler
	server                      *dgrpc.Server
}

// TODO don't depend on both....
func NewInjector(
	ctx context.Context,
	handler Handler,
	blockstreamAddr string,
	blockStore dstore.Store,
	preprocessorThreadCount int,
	parallelDownloadThreadCount int,
) *Injector {
	return &Injector{
		ctx:             ctx,
		handler:         handler,
		blockstreamAddr: blockstreamAddr,
		blockStore:      blockStore,
		Shutter:         shutter.New(),
		slotMetrics: slotMetrics{
			startTime: time.Now(),
		},
		preprocessorThreadCount:     preprocessorThreadCount,
		parallelDownloadThreadCount: parallelDownloadThreadCount,
	}
}

func (i *Injector) SetupSource(startBlockNum uint64, ignoreCheckpointOnLaunch bool) error {
	zlog.Info("setting up serumhist source",
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

	i.OnTerminating(func(err error) {
		zlog.Info("shutting down injector, attempting to close underlying handler")
		if err := i.handler.Close(); err != nil {
			zlog.Error("error closing underlying serumhist injector handler", zap.Error(err))
		}
	})

	//if i.grpcAddr != "" {
	//	server := dgrpc.NewServer2(dgrpc.WithLogger(zlog))
	//	server.RegisterService(func(gs *grpc.Server) {
	//		pbserumhist.RegisterSerumOrderTrackerServer(gs, i)
	//		pbhealth.RegisterHealthServer(gs, i)
	//	})
	//
	//	zlog.Info("listening for serum history",
	//		zap.String("addr", i.grpcAddr),
	//	)
	//
	//	i.OnTerminating(func(err error) {
	//		server.Shutdown(30 * time.Second)
	//	})
	//
	//	go server.Launch(i.grpcAddr)
	//}

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
