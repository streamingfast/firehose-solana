package pbcodec

import (
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/golang/protobuf/proto"
)

func (s *Slot) ID() string {
	return s.Id
}

func (s *Slot) Num() uint64 {
	return s.Number
}

func (s *Slot) Split(removeFromInstruction bool) *AccountChangesBundle {
	bundle := &AccountChangesBundle{}
	for _, trx := range s.Transactions {
		bundleTransaction := &AccountChangesPerTrxIndex{}
		for _, instruction := range trx.Instructions {
			bundleInstruction := &AccountChangesPerInstruction{}
			for _, change := range instruction.AccountChanges {
				bundleInstruction.Changes = append(bundleInstruction.Changes, change)
			}
			bundleTransaction.Instructions = append(bundleTransaction.Instructions, bundleInstruction)
			if removeFromInstruction {
				instruction.AccountChanges = nil
			}
		}
		bundle.Transactions = append(bundle.Transactions, bundleTransaction)
	}
	return bundle
}

func (s *Slot) Join(bundle *AccountChangesBundle) {
	for ti, bundleTransaction := range bundle.Transactions {
		for ii, bundleInstruction := range bundleTransaction.Instructions {
			s.Transactions[ti].Instructions[ii].AccountChanges = bundleInstruction.Changes
		}
	}
}

func (m *Block) PreviousID() string {
	return m.PreviousId
}

func (m *Block) Time() time.Time {
	return time.Unix(int64(m.GenesisUnixTimestamp), 0)
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
