package codec

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/bstream"
	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
)

// FIXME: Solana protocol will be the value 3, might not work everywehre ... we will see!
var Protocol_SOL = pbbstream.Protocol(3)

func BlockDecoder(blk *bstream.Block) (interface{}, error) {
	if blk.Kind() != Protocol_SOL {
		return nil, fmt.Errorf("expected kind %s, got %s", Protocol_SOL, blk.Kind())
	}

	if blk.Version() != 1 {
		return nil, fmt.Errorf("this decoder only knows about version 1, got %d", blk.Version())
	}

	block := new(pbcodec.Block)
	payload, err := blk.Payload.Get()
	if err != nil {
		return nil, fmt.Errorf("getting payload: %s", err)
	}

	err = proto.Unmarshal(payload, block)
	if err != nil {
		return nil, fmt.Errorf("unable to decode payload: %s", err)
	}

	// This whole BlockDecoder method is being called through the `bstream.Block.ToNative()`
	// method. Hence, it's a great place to add temporary data normalization calls to backport
	// some features that were not in all blocks yet (because we did not re-process all blocks
	// yet).
	//
	// Thoughts for the future: Ideally, we would leverage the version information here to take
	// a decision, like `do X if version <= 2.1` so we would not pay the performance hit
	// automatically instead of having to re-deploy a new version of bstream (which means
	// rebuild everything mostly)

	return block, nil
}
