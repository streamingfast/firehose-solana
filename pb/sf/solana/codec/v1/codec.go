package pbcodec

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	"google.golang.org/protobuf/proto"
)

func (b *Block) ID() string {
	return base58.Encode(b.Id)
}

func (b *Block) Num() uint64 {
	return b.Number
}

// TODO: move to `codec/`...
func (b *Block) Split(removeFromInstruction bool) *AccountChangesBundle {
	bundle := &AccountChangesBundle{}

	for _, trx := range b.Transactions {
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

func (b *Block) Join(ctx context.Context, notFoundFunc func(fileName string) bool) error {
	bundle, err := b.Retrieve(ctx, notFoundFunc)
	if err != nil {
		return fmt.Errorf("error retrieving account changes: %w", err)
	}

	b.join(bundle)
	return nil
}

func (b *Block) join(bundle *AccountChangesBundle) {
	for ti, bundleTransaction := range bundle.Transactions {
		for ii, bundleInstruction := range bundleTransaction.Instructions {
			b.Transactions[ti].Instructions[ii].AccountChanges = bundleInstruction.Changes
		}
	}
}

func (b *Block) Retrieve(ctx context.Context, notFoundFunc func(fileName string) bool) (*AccountChangesBundle, error) {
	store, filename, err := dstore.NewStoreFromURL(b.AccountChangesFileRef,
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

func (b *Block) PreviousID() string {
	return hex.EncodeToString(b.PreviousId)
}

func (b *Block) Time() time.Time {
	return time.Unix(int64(b.GenesisUnixTimestamp), 0)
}

func (b *Block) LIBNum() uint64 {
	return b.RootNum
}

func (b *Block) AsRef() bstream.BlockRef {
	return bstream.NewBlockRef(b.ID(), b.Number)
}

//func (te *TransactionError) DecodedPayload() proto.Message {
//	var x ptypes.DynamicAny
//	if err := ptypes.UnmarshalAny(te.Payload, &x); err != nil {
//		panic(fmt.Sprintf("unable to unmarshall transaction error payload: %s", err))
//	}
//	return x.Message
//}
//
//func (ie *InstructionError) DecodedPayload() proto.Message {
//	var x ptypes.DynamicAny
//	if err := ptypes.UnmarshalAny(ie.Payload, &x); err != nil {
//		panic(fmt.Sprintf("unable to unmarshall instruction error payload: %s", err))
//	}
//	return x.Message
//}

func BlockToBuffer(block *Block) ([]byte, error) {
	buf, err := proto.Marshal(block)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func MustBlockToBuffer(block *Block) []byte {
	buf, err := BlockToBuffer(block)
	if err != nil {
		panic(err)
	}
	return buf
}
