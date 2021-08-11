package kvloader

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/streamingfast/kvdb/store"
	pbserumhist "github.com/streamingfast/sf-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/streamingfast/sf-solana/serumhist/keyer"
)

func (kv *KVLoader) GetCheckpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {
	key := keyer.EncodeCheckpoint()

	ctx, cancel := context.WithTimeout(ctx, DatabaseTimeout)
	defer cancel()

	val, err := kv.kvdb.Get(ctx, key)
	if err == store.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error while reading checkpoint: %w", err)
	}

	out := &pbserumhist.Checkpoint{}
	if err := proto.Unmarshal(val, out); err != nil {
		return nil, err
	}

	return out, nil
}
