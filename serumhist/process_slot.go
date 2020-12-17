package serumhist

import (
	"fmt"

	bin "github.com/dfuse-io/binary"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	kvdb "github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go/diff"
	"github.com/dfuse-io/solana-go/programs/serum"
	"go.uber.org/zap"
)

func (l *Loader) PutSlot(slot *pbcodec.Slot) error {
	if traceEnabled {
		zlog.Debug("processing slot", zap.String("slot_id", slot.Id))
	}

	if err := l.processSerumSlot(slot); err != nil {
		return fmt.Errorf("put slot: unable to process serum slot: %w", err)
	}

	return nil
}

func (l *Loader) processSerumSlot(slot *pbcodec.Slot) error {
	for _, transaction := range slot.Transactions {
		for _, instruction := range transaction.Instructions {
			for _, accountChange := range instruction.AccountChanges {
				shouldProcess, processor := l.shouldProcessAccountChange(accountChange)
				if !shouldProcess {
					continue
				}

				_, err := processor(accountChange)
				if err != nil {
					zlog.Warn("error processing account changes",
						zap.String("account_key", accountChange.Pubkey),
						zap.String("error", err.Error()),
					)
					continue
				}

			}
		}
	}
	return nil
}

type processor = func(accountChange *pbcodec.AccountChange) ([]kvdb.KV, error)

func (l *Loader) shouldProcessAccountChange(accountChange *pbcodec.AccountChange) (bool, processor) {
	var f *serum.AccountFlag
	if err := bin.NewDecoder(accountChange.PrevData).Decode(&f); err != nil {
		zlog.Warn("unable to decode account flag",
			zap.String("account_key", accountChange.Pubkey),
			zap.String("error", err.Error()),
		)
		return false, nil
	}
	if !f.Is(serum.AccountFlagInitialized) {
		return false, nil
	}
	if f.Is(serum.AccountFlagRequestQueue) {
		return true, l.processRequestQueueAccountChange
	}
	if f.Is(serum.AccountFlagEventQueue) {
		return true, l.processEventQueue
	}

	return false, nil
}

func (l *Loader) processRequestQueueAccountChange(accountChange *pbcodec.AccountChange) (out []kvdb.KV, err error) {
	var oldData *serum.RequestQueue
	if err := bin.NewDecoder(accountChange.PrevData).Decode(&oldData); err != nil {
		return out, fmt.Errorf("unable to decode 'event queue' old data: %w", err)
	}

	var newData *serum.RequestQueue
	if err := bin.NewDecoder(accountChange.NewData).Decode(&newData); err != nil {
		return out, fmt.Errorf("unable to decode 'event queue' new data: %w", err)
	}
	return l.getRequestQueueChangeKeys(oldData, newData), nil
}

func (l *Loader) processEventQueue(accountChange *pbcodec.AccountChange) (out []kvdb.KV, err error) {
	var oldData *serum.EventQueue
	if err := bin.NewDecoder(accountChange.PrevData).Decode(&oldData); err != nil {
		return out, fmt.Errorf("unable to decode 'request queue' old data: %w", err)
	}

	var newData *serum.EventQueue
	if err := bin.NewDecoder(accountChange.NewData).Decode(&newData); err != nil {
		return out, fmt.Errorf("unable to decode 'request queue' new data: %w", err)
	}

	return l.getEventQueueChangeKeys(oldData, newData), nil
}

func (l *Loader) getRequestQueueChangeKeys(old, new *serum.RequestQueue) (out []kvdb.KV) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		fmt.Println("RequestQueue " + event.String())
		fmt.Println("Path " + event.Path.String())
	}))
	return
}

func (l *Loader) getEventQueueChangeKeys(old, new *serum.EventQueue) (out []kvdb.KV) {
	diff.Diff(old, new, diff.OnEvent(func(event diff.Event) {
		fmt.Println("EventQueue " + event.String())
		fmt.Println("Path " + event.Path.String())
	}))
	return
}
