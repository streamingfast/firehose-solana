package tools

import (
	"fmt"
	"testing"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	_ "github.com/dfuse-io/dfuse-solana/codec"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/test-go/testify/require"
)

func TestBlockNum_String(t *testing.T) {
	handlerFunc := bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
		slot := block.ToNative().(*pbcodec.Slot)
		if slot.Number == 64618790 {
			return fmt.Errorf("end block")
		}

		forkObj := obj.(*forkable.ForkableObject)
		fmt.Printf("Slot %s step %s (prev %s) (lib %d)\n", block.String(), forkObj.Step.String(), block.PreviousId, slot.LIBNum())
		return nil
	})

	handlerGate := bstream.NewBlockNumGate(64618760, bstream.GateInclusive, handlerFunc)

	forkableOptions := []forkable.Option{
		forkable.WithFilters(forkable.StepsAll),
	}

	forkHandler := forkable.New(handlerGate, forkableOptions...)

	store, err := dstore.NewDBinStore("gs://dfuseio-global-blocks-us/sol-mainnet/v1")
	require.NoError(t, err)

	source := bstream.NewFileSource(
		store,
		64618700,
		1,
		nil,
		forkHandler,
	)

	source.Run()
	fmt.Println("end")
}
