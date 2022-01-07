package codec

import (
	"fmt"

	"github.com/streamingfast/bstream"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"google.golang.org/protobuf/proto"
)

func BlockFromProto(slot *pbcodec.Block) (*bstream.Block, error) {
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
		LibNum:         slot.LIBNum(),
		PayloadKind:    Protocol_SOL,
		PayloadVersion: int32(slot.Version),
	}
	return bstream.GetBlockPayloadSetter(block, content)
}
