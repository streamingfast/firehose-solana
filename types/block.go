package types

import (
	"fmt"
	"time"

	"github.com/streamingfast/bstream"
	pbsolv1 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v1"
	pbsolv2 "github.com/streamingfast/firehose-solana/types/pb/sf/solana/type/v2"
	"google.golang.org/protobuf/proto"
)

func BlockFromPBSolProto(slot *pbsolv2.Block) (*bstream.Block, error) {
	blockTime := slot.Time()

	content, err := proto.Marshal(slot)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal to binary form: %s", err)
	}

	block := &bstream.Block{
		Id:             slot.ID(),
		Number:         slot.Num(),
		PreviousId:     slot.PreviousID(),
		Timestamp:      blockTime,
		PayloadKind:    Protocol_SOL,
		PayloadVersion: int32(slot.Version),
	}
	return bstream.GetBlockPayloadSetter(block, content)
}

func BlockFromPBSolanaProto(blk *pbsolv1.Block) (*bstream.Block, error) {
	content, err := proto.Marshal(blk)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal to binary form: %s", err)
	}
	blockTime := time.Unix(blk.BlockTime.GetTimestamp(), 0)
	block := &bstream.Block{
		Id:             blk.Blockhash,
		Number:         blk.Slot,
		PreviousId:     blk.PreviousBlockhash,
		LibNum:         blk.ParentSlot,
		Timestamp:      blockTime,
		PayloadKind:    Protocol_SOL,
		PayloadVersion: 1,
	}
	return bstream.GetBlockPayloadSetter(block, content)
}
