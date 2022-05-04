package types

import (
	"fmt"
	"time"

	pbsolana "github.com/streamingfast/sf-solana/types/pb/sol/type/v1"

	pbsol "github.com/streamingfast/sf-solana/types/pb/sf/solana/type/v1"

	"github.com/streamingfast/bstream"
	"google.golang.org/protobuf/proto"
)

func BlockFromPBSolProto(slot *pbsol.Block) (*bstream.Block, error) {
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

func BlockFromPBSolanaProto(blk *pbsolana.ConfirmedBlock) (*bstream.Block, error) {
	content, err := proto.Marshal(blk)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal to binary form: %s", err)
	}

	blockHeight := uint64(0)
	if blk.BlockHeight != nil {
		blockHeight = blk.BlockHeight.GetBlockHeight()
	}
	blockTime := time.Unix(blk.BlockTime.GetTimestamp(), 0)
	block := &bstream.Block{
		Id:             blk.Blockhash,
		Number:         blockHeight,
		PreviousId:     blk.PreviousBlockhash,
		Timestamp:      blockTime,
		PayloadKind:    Protocol_SOL,
		PayloadVersion: 1,
	}
	return bstream.GetBlockPayloadSetter(block, content)
}
