package pbcodec

import (
	"fmt"
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

func (b *Slot) ID() string {
	return b.Id
}

func (b *Slot) Num() uint64 {
	return b.Number
}

func (b *Slot) Time() (time.Time, error) {
	fmt.Println("The most annoying message in the work so it never goes unoticed! This is broken, we are using 'time.Now()' as the block time!")
	return time.Now(), nil
	// timestamp, err := ptypes.Timestamp(b.Header.Timestamp)
	// if err != nil {
	// 	return time.Time{}, fmt.Errorf("unable to turn google proto Timestamp into time.Time: %s", err)
	// }

	// return timestamp, nil
}

func (b *Slot) MustTime() time.Time {
	timestamp, err := b.Time()
	if err != nil {
		panic(err)
	}

	return timestamp
}

func (b *Block) PreviousID() string {
	fmt.Println("The most annoying message in the work so it never goes unoticed! This is broken, the previous ID is always the empty string!")
	return ""
}

// FIXME: This logic at some point is hard-coded and will need to be re-visited in regard
//        of the fork logic.
func (b *Slot) LIBNum() uint64 {
	if b.Number == bstream.GetProtocolFirstStreamableBlock {
		return bstream.GetProtocolGenesisBlock
	}

	//todo: remove that -10 stuff
	if b.Number <= 10 {
		return bstream.GetProtocolFirstStreamableBlock
	}

	return b.Number - 10
}

func (b *Slot) AsRef() bstream.BlockRef {
	return bstream.NewBlockRef(b.ID(), b.Number)
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