package serumhist

import (
	"context"

	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	pbaccounthist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"google.golang.org/protobuf/proto"
)

func (i *Injector) writeCheckpoint(ctx context.Context, slot *pbcodec.Slot) error {

	key := keyer.EncodeCheckpoint()

	checkpoint := &pbaccounthist.Checkpoint{
		LastWrittenSlotNum: slot.Number,
		LastWrittenLostId:  slot.Id,
	}

	value, err := proto.Marshal(checkpoint)
	if err != nil {
		return err
	}

	return i.kvdb.Put(ctx, key, value)
}
