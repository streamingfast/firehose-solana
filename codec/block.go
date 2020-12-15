package codec

import (
	"fmt"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/golang/protobuf/proto"
)

func BlockFromProto(slot *pbcodec.Slot) (*bstream.Block, error) {
	blockTime, err := slot.Time()
	if err != nil {
		return nil, err
	}

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