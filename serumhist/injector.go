package serumhist

import (
	"context"
	"fmt"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/keepalive"
	"io"
	"time"

	"github.com/dfuse-io/bstream"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"
	"github.com/dfuse-io/kvdb/store"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/dfuse-io/shutter"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/serum"
	"go.uber.org/zap"
)

type Injector struct {
	*shutter.Shutter
	kvdb              store.KVStore
	flushSlotInterval uint64
	lastTickBlock     uint64
	lastTickTime      time.Time
	blockStreamV2Addr string
	blockstreamAddr   string
	healthy           bool
	firehoseClient    pbbstream.BlockStreamV2Client
	blockStreamClient pbbstream.BlockStreamClient // temp used to

	eventQueues  map[string]solana.PublicKey
	requesQueues map[string]solana.PublicKey
}

func NewInjector(
	blockstreamV2Addr string,
	blockstreamAddr string,
	kvdb store.KVStore,
	flushSlotInterval uint64,
) *Injector {
	return &Injector{
		blockStreamV2Addr: blockstreamV2Addr,
		blockstreamAddr:   blockstreamAddr,
		Shutter:           shutter.New(),
		flushSlotInterval: flushSlotInterval,
		eventQueues:       map[string]solana.PublicKey{},
		requesQueues:      map[string]solana.PublicKey{},
		kvdb:              kvdb,
	}
}

func (l *Injector) Setup() error {

	conn, err := grpc.Dial(
		l.blockstreamAddr,
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // send pings every (x seconds) there is no activity
			Timeout:             10 * time.Second, // wait that amount of time for ping ack before considering the connection dead
			PermitWithoutStream: true,             // send pings even without active streams
		}),
		grpc.WithDefaultCallOptions([]grpc.CallOption{grpc.MaxCallRecvMsgSize(1024 * 1024 * 300), grpc.WaitForReady(true)}...),
	)

	markets, err := serum.KnownMarket()
	if err != nil {
		return fmt.Errorf("unable to retrieve known markets: %w", err)
	}

	for _, market := range markets {
		l.eventQueues[market.MarketV2.EventQueue.String()] = market.Address
		l.requesQueues[market.MarketV2.RequestQueue.String()] = market.Address
	}

	l.blockStreamClient = pbbstream.NewBlockStreamClient(conn)
	return nil
}

func (l *Injector) Launch(ctx context.Context, startBlockNum uint64) error {
	//req := &pbbstream.BlocksRequestV2{
	//	StartBlockNum:     int64(startBlockNum),
	//	ExcludeStartBlock: true,
	//	Decoded:           true,
	//	HandleForks:       true,
	//	HandleForksSteps: []pbbstream.ForkStep{
	//		pbbstream.ForkStep_STEP_IRREVERSIBLE,
	//	},
	//}
	req := &pbbstream.BlockRequest{
		Burst:       100,
		ContentType: "sol",
		Requester:   "serumhist",
	}
	zlog.Info("launching serumdb loader",
		zap.Reflect("blockstream_request", req),
	)

	// executor, err := l.firehoseClient.Blocks(ctx, req)
	executor, err := l.blockStreamClient.Blocks(ctx, req)
	if err != nil {
		return fmt.Errorf("unable to setup block stream client: %w", err)
	}
	{
		msg, err := executor.Recv()
		if err == io.EOF {
			zlog.Info("received EOF in listening stream, expected a long-running stream here")
			return nil
		}
		if err != nil {
			return err
		}

		l.setHealthy()

		blk, err := bstream.BlockFromProto(msg)
		if err != nil {
			return fmt.Errorf("unable to transform to bstream.Block: %w", err)
		}
		slot := blk.ToNative().(*pbcodec.Slot)

		//if msg.Undo {
		//	return fmt.Errorf("blockstreamv2 should never send undo signals, irreversible only please")
		//}
		//
		//if msg.Step != pbbstream.ForkStep_STEP_IRREVERSIBLE {
		//	return fmt.Errorf("blockstreamv2 should never pass something that is not irreversible")
		//}

		//if slot.Number%100 == 0 {
		zlog.Info("processed slot 1",
			zap.Uint64("slot_number", slot.Number),
			zap.String("slot_id", slot.Id),
			zap.String("previous_id", slot.PreviousId),
			zap.Uint32("transaction_count", slot.TransactionCount),
		)
		//}

		l.ProcessSlot(ctx, slot)

		if err := l.writeCheckpoint(ctx, slot); err != nil {
			return fmt.Errorf("error while saving block checkpoint")
		}

		if err := l.flush(ctx, slot); err != nil {
			return fmt.Errorf("error while flushing: %w", err)
		}

		t, err := slot.Time()
		if err != nil {
			return fmt.Errorf("unable to resolve slot time for slot %q: %w", slot.Number, err)
		}

		err = l.FlushIfNeeded(slot.Number, t)
		if err != nil {
			zlog.Error("flushIfNeeded", zap.Error(err))
			return err
		}
	}
	return nil
}

func (l *Injector) DoFlush(slotNum uint64, reason string) error {
	zlog.Debug("flushing block",
		zap.Uint64("slot_num", slotNum),
		zap.String("reason", reason),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	err := l.kvdb.FlushPuts(ctx)
	if err != nil {
		return fmt.Errorf("db flush: %w", err)
	}
	return nil
}

func (l *Injector) FlushIfNeeded(slotNum uint64, slotTime time.Time) error {
	batchSizeReached := slotNum%l.flushSlotInterval == 0
	closeToHeadBlockTime := time.Since(slotTime) < 25*time.Second

	if batchSizeReached || closeToHeadBlockTime {
		reason := "needed"
		if batchSizeReached {
			reason += ", batch size reached"
		}

		if closeToHeadBlockTime {
			reason += ", close to head block"
		}

		err := l.DoFlush(slotNum, reason)
		if err != nil {
			return err
		}
		metrics.HeadBlockNumber.SetUint64(slotNum)
	}

	return nil
}
