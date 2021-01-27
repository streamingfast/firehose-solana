package pbcodec

import (
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/golang/protobuf/proto"
)

// func MustBlockRefAsProto(ref bstream.BlockRef) *BlockRef {
// 	if ref == nil || bstream.EqualsBlockRefs(ref, bstream.BlockRefEmpty) {
// 		return nil
// 	}

// 	hash, err := hex.DecodeString(ref.ID())
// 	if err != nil {
// 		panic(fmt.Errorf("invalid block hash %q: %w", ref.ID(), err))
// 	}

// 	return &BlockRef{
// 		Hash:   hash,
// 		Number: ref.Num(),
// 	}
// }

// func (b *BlockRef) AsBstreamBlockRef() bstream.BlockRef {
// 	return bstream.NewBlockRef(hex.EncodeToString(b.Hash), b.Number)
// }

// TODO: We should probably memoize all fields that requires computation
//       like ID() and likes.

func (s *Slot) ID() string {
	return s.Id
}

func (s *Slot) Num() uint64 {
	return s.Number
}

func (m *Block) PreviousID() string {
	return m.PreviousId
}

func (m *Block) Time() time.Time {
	return time.Unix(int64(m.ClockUnixTimestamp), 0)
}

// FIXME: This logic at some point is hard-coded and will need to be re-visited in regard
//        of the fork logic.
func (s *Slot) LIBNum() uint64 {
	if s.Number == bstream.GetProtocolFirstStreamableBlock {
		return bstream.GetProtocolGenesisBlock
	}

	//todo: remove that -10 stuff
	if s.Number <= 10 {
		return bstream.GetProtocolFirstStreamableBlock
	}

	return s.Number - 10
}

func (s *Slot) AsRef() bstream.BlockRef {
	return bstream.NewBlockRef(s.ID(), s.Number)
}

func BlockToBuffer(block *Slot) ([]byte, error) {
	buf, err := proto.Marshal(block)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func MustBlockToBuffer(block *Slot) []byte {
	buf, err := BlockToBuffer(block)
	if err != nil {
		panic(err)
	}
	return buf
}
