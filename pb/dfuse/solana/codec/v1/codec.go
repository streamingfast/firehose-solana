package pbcodec

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	"github.com/golang/protobuf/proto"
)

func (s *Slot) ID() string {
	return s.Id
}

func (s *Slot) Num() uint64 {
	return s.Number
}

// TODO: move to `codec/`...
func (s *Slot) Split(removeFromInstruction bool) *AccountChangesBundle {
	bundle := &AccountChangesBundle{}

	for _, trx := range s.Transactions {
		bundleTransaction := &AccountChangesPerTrxIndex{
			TrxId: trx.Id,
		}
		for _, instruction := range trx.Instructions {
			bundleTransaction.Instructions = append(
				bundleTransaction.Instructions,
				&AccountChangesPerInstruction{
					Changes: instruction.AccountChanges,
				})
			if removeFromInstruction {
				instruction.AccountChanges = nil
			}
		}
		bundle.Transactions = append(bundle.Transactions, bundleTransaction)
	}
	return bundle
}

func (s *Slot) Join(ctx context.Context, notFoundFunc func(fileName string) bool) error {
	bundle, err := s.Retrieve(ctx, notFoundFunc)
	if err != nil {
		return fmt.Errorf("error retrieving account changes: %w", err)
	}

	s.join(bundle)
	return nil
}

func (s *Slot) join(bundle *AccountChangesBundle) {
	for ti, bundleTransaction := range bundle.Transactions {
		for ii, bundleInstruction := range bundleTransaction.Instructions {
			s.Transactions[ti].Instructions[ii].AccountChanges = bundleInstruction.Changes
		}
	}
}

func (s *Slot) Retrieve(ctx context.Context, notFoundFunc func(fileName string) bool) (*AccountChangesBundle, error) {
	store, filename, err := dstore.NewStoreFromURL(s.AccountChangesFileRef,
		dstore.Compression("zstd"),
	)
	if err != nil {
		return nil, fmt.Errorf("store from url: %s: %w", filename, err)
	}

	for {
		exist, err := store.FileExists(ctx, filename)
		if err != nil {
			return nil, fmt.Errorf("file exist: %s : %w", filename, err)
		}
		if !exist {
			if notFoundFunc(filename) {
				//notFoundFunc should sleep and return true to retry
				continue
			}
			return nil, fmt.Errorf("retry break by not found func: %s", filename)
		}

		break
	}

	reader, err := store.OpenObject(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("open object: %s : %w", filename, err)
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read all: %s : %w", filename, err)
	}

	bundle := &AccountChangesBundle{}
	err = proto.Unmarshal(data, bundle)
	if err != nil {
		return nil, fmt.Errorf("proto unmarshal: %s : %w", filename, err)
	}

	return bundle, nil
}

func (m *Block) PreviousID() string {
	return m.PreviousId
}

func (m *Block) Time() time.Time {
	return time.Unix(int64(m.GenesisUnixTimestamp), 0)
}

func (s *Slot) LIBNum() uint64 {
	return s.Block.RootNum
}

func (s *Slot) AsRef() bstream.BlockRef {
	return bstream.NewBlockRef(s.ID(), s.Number)
}

func (te *TransactionError) DecodedPayload() proto.Message {
	var x ptypes.DynamicAny
	if err := ptypes.UnmarshalAny(te.Payload, &x); err != nil {
		panic(fmt.Sprintf("unable to unmarshall transaction error payload: %s", err))
	}
	return x.Message
}

func (ie *InstructionError) DecodedPayload() proto.Message {
	var x ptypes.DynamicAny
	if err := ptypes.UnmarshalAny(ie.Payload, &x); err != nil {
		panic(fmt.Sprintf("unable to unmarshall instruction error payload: %s", err))
	}
	return x.Message
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
