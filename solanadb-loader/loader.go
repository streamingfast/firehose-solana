package solanadb_loader

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dfuse-solana/solanadb"
	"github.com/dfuse-io/dfuse-solana/solanadb-loader/metrics"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/dfuse-io/shutter"
	"github.com/golang/protobuf/ptypes"
	"go.opencensus.io/plugin/ocgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Loader struct {
	*shutter.Shutter
	db                solanadb.DBWriter
	batchSize         uint64
	lastTickBlock     uint64
	lastTickTime      time.Time
	blockstreamV2Addr string
	source            bstream.Source
	healthy           bool
	blockStreamClient pbbstream.BlockStreamV2Client
}

func SetupLoader(
	blockstreamV2Addr string,
	db solanadb.DB,
	batchSize uint64,
) (*Loader, error) {

	conn, err := grpc.Dial(
		blockstreamV2Addr,
		grpc.WithInsecure(),
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
	)
	if err != nil {
		return nil, fmt.Errorf("unexpected block stream dial connection failure: %w", err)
	}

	loader := &Loader{
		blockstreamV2Addr: blockstreamV2Addr,
		Shutter:           shutter.New(),
		batchSize:         batchSize,
		blockStreamClient: pbbstream.NewBlockStreamV2Client(conn),
		db:                db,
	}
	return loader, nil
}

func (l *Loader) Launch(ctx context.Context, startBlockNum uint64) error {
	req := &pbbstream.BlocksRequestV2{
		StartBlockNum:     int64(startBlockNum),
		ExcludeStartBlock: true,
		Decoded:           true,
		HandleForks:       true,
		HandleForksSteps: []pbbstream.ForkStep{
			pbbstream.ForkStep_STEP_IRREVERSIBLE,
		},
	}
	zlog.Info("launching serumdb loader",
		zap.Reflect("blockstream_request", req),
	)

	executor, err := l.blockStreamClient.Blocks(ctx, req)
	if err != nil {
		return fmt.Errorf("")
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

		slot := &pbcodec.Slot{}
		if err := ptypes.UnmarshalAny(msg.Block, slot); err != nil {
			return fmt.Errorf("decoding any of type %q: %w", msg.Block.TypeUrl, err)
		}

		if msg.Undo {
			return fmt.Errorf("blockstreamv2 should never send undo signals, irreversible only please!")
		}

		if msg.Step != pbbstream.ForkStep_STEP_IRREVERSIBLE {
			return fmt.Errorf("blockstreamv2 should never pass something that is not irreversible")
		}

		if slot.Number%100 == 0 {
			zlog.Info("Processed slot 1/100",
				zap.Uint64("slot_number", slot.Number),
				zap.String("slot_id", slot.Id),
				zap.String("previous_id", slot.PreviousId),
				zap.Uint32("transaction_count", slot.TransactionCount),
			)
		}

		l.db.PutSlot(ctx, slot)

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

func (l *Loader) DoFlush(slotNum uint64, reason string) error {
	zlog.Debug("flushing block",
		zap.Uint64("slot_num", slotNum),
		zap.String("reason", reason),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	err := l.db.Flush(ctx)
	if err != nil {
		return fmt.Errorf("db flush: %w", err)
	}
	return nil
}

func (l *Loader) FlushIfNeeded(slotNum uint64, slotTime time.Time) error {
	batchSizeReached := slotNum%l.batchSize == 0
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
