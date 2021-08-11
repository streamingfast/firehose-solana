package codec

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/bstream"
	pbcodec "github.com/streamingfast/sf-solana/pb/dfuse/solana/codec/v1"
)

func BlockFromProto(slot *pbcodec.Slot) (*bstream.Block, error) {
	blockTime := slot.Block.Time()

	content, err := proto.Marshal(slot)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal to binary form: %s", err)
	}

	return &bstream.Block{
		Id:             slot.ID(),
		Number:         slot.Num(),
		PreviousId:     slot.PreviousId,
		Timestamp:      blockTime,
		LibNum:         slot.LIBNum(),
		PayloadKind:    Protocol_SOL,
		PayloadVersion: int32(slot.Version),
		PayloadBuffer:  content,
	}, nil
}
