package serumhist

import (
	"fmt"

	"github.com/dfuse-io/solana-go/programs/serum"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"go.uber.org/zap"
)

type serumInstruction struct {
	trxIdx     uint64
	instIdx    uint64
	trxtID     string
	accChanges []*pbcodec.AccountChange
	native     *serum.Instruction
}

func (i *Injector) ProcessBlock(blk *bstream.Block, obj interface{}) error {
	i.setHealthy()

	slot := blk.ToNative().(*pbcodec.Slot)

	if slot.Number%logEveryXSlot == 0 {
		zlog.Info(fmt.Sprintf("processed %d slot", logEveryXSlot),
			zap.Uint64("slot_number", slot.Number),
			zap.String("slot_id", slot.Id),
			zap.String("previous_id", slot.PreviousId),
			zap.Uint32("transaction_count", slot.TransactionCount),
		)
	}

	for _, inst := range obj.([]*serumInstruction) {
		if err := i.processInstruction(i.ctx, slot.Number, slot.Block.Time(), inst); err != nil {
			return fmt.Errorf("process serum instruction: %w", err)
		}

	}

	if err := i.writeCheckpoint(i.ctx, slot); err != nil {
		return fmt.Errorf("error while saving block checkpoint")
	}

	if err := i.flush(i.ctx, slot); err != nil {
		return fmt.Errorf("error while flushing: %w", err)
	}

	t := slot.Block.Time()

	err := i.flushIfNeeded(slot.Number, t)
	if err != nil {
		zlog.Error("flushIfNeeded", zap.Error(err))
		return err
	}

	return nil
}
