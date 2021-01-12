package serumhist

import (
	"context"
	"time"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"
)

func (i *Injector) flush(ctx context.Context, slot *pbcodec.Slot) error {
	slotNum := slot.Number
	closeToHeadBlockTime := false
	onFlushIntervalBoundary := slotNum%i.flushSlotInterval == 0

	t := slot.Block.Time()
	closeToHeadBlockTime = time.Since(t) < 25*time.Second

	if onFlushIntervalBoundary || closeToHeadBlockTime {
		reason := "needed"
		if onFlushIntervalBoundary {
			reason += ", flush interval boundary reached"
		}

		if closeToHeadBlockTime {
			reason += ", close to head block"
		}

		err := i.DoFlush(slotNum, reason)
		if err != nil {
			return err
		}
		metrics.HeadBlockNumber.SetUint64(slotNum)
	}

	return nil
}
