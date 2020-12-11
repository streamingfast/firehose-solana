package codec

import (
	"fmt"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/golang/protobuf/proto"
)

func BlockFromProto(blk *pbcodec.Block) (*bstream.Block, error) {
	blockTime, err := blk.Time()
	if err != nil {
		return nil, err
	}

	content, err := proto.Marshal(blk)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal to binary form: %s", err)
	}

	return &bstream.Block{
		Id:          blk.ID(),
		Number:      blk.Num(),
		PreviousId:  blk.PreviousID(),
		Timestamp:   blockTime,
		LibNum:      blk.LIBNum(),
		PayloadKind: Protocol_SOL,
		// PayloadKind:    pbbstream.Protocol_SOL,
		PayloadVersion: int32(blk.Version),
		PayloadBuffer:  content,
	}, nil
}
