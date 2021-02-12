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
	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Injector struct {
	*shutter.Shutter
	ctx                     context.Context
	kvdb                    store.KVStore
	flushSlotInterval       uint64
	lastTickBlock           uint64
	lastTickTime            time.Time
	blockStore              dstore.Store
	blockstreamAddr         string
	healthy                 bool
	cache                   *tradingAccountCache
	source                  *firehose.Firehose
	slotMetrics             slotMetrics
	preprocessorThreadCount int
}

func NewInjector(
	ctx context.Context,
	blockstreamAddr string,
	blockStore dstore.Store,
	kvdb store.KVStore,
	flushSlotInterval uint64,
	preprocessorThreadCount int,
) *Injector {
	return &Injector{
		ctx:               ctx,
		blockstreamAddr:   blockstreamAddr,
		blockStore:        blockStore,
		Shutter:           shutter.New(),
		flushSlotInterval: flushSlotInterval,
		kvdb:              kvdb,
		cache:             newTradingAccountCache(kvdb),
		slotMetrics: slotMetrics{
			startTime: time.Now(),
		},
		preprocessorThreadCount: preprocessorThreadCount,
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
		firehose.WithPreproc(i.preprocessSlot),
		firehose.WithLogger(zlog),
		firehose.WithForkableSteps(forkable.StepNew | forkable.StepIrreversible),
		firehose.WithConcurrentPreprocessor(i.preprocessorThreadCount),
	}

	if i.blockstreamAddr != "" {
		liveStreamFactory := bstream.SourceFactory(func(subHandler bstream.Handler) bstream.Source {
			return blockstream.NewSource(
				i.ctx,
				i.blockstreamAddr,
				200,
				subHandler,
			)
		})
		options = append(options, firehose.WithLiveSource(liveStreamFactory))
	}

	fhose := firehose.New(
		[]dstore.Store{i.blockStore},
		int64(checkpoint.LastWrittenSlotNum),
		i,
		options...,
	)
	i.source = fhose
	return nil
}

func (i *Injector) Launch() error {
	zlog.Info("launching serumhist injector")
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

	err := i.kvdb.FlushPuts(ctx)
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
