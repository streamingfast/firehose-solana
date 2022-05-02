package types

import (
	"fmt"

	pbsol "github.com/streamingfast/sf-solana/types/pb/sf/solana/type/v1"

	"github.com/streamingfast/bstream"
	"google.golang.org/protobuf/proto"
)

func BlockFromProto(slot *pbsol.Block) (*bstream.Block, error) {
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
