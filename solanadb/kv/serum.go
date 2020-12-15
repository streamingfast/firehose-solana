package kv

import (
	"context"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
)

func (db *DB) processSerumSlot(ctx context.Context, slot *pbcodec.Slot) error {
	for _, transactionTrace := range slot.TransactionTraces {
		for _, instruction := range transactionTrace.InstructionTraces {
			for _, accountChange := range instruction.AccountChanges {
				_ = accountChange
				if traceEnabled {
					zlog.Info("processing info")
				}
			}
		}
	}

	return nil
}
