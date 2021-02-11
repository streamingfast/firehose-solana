package tools

import (
	"fmt"
	"testing"

	"go.uber.org/zap"

	"github.com/dfuse-io/logging"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	_ "github.com/dfuse-io/dfuse-solana/codec"
	"github.com/dfuse-io/dstore"
	"github.com/test-go/testify/require"
)

func init() {
	logging.TestingOverride()
}

func TestBlockNum_String(t *testing.T) {
	t.Skip("block file debugging.. not really a test... just for fun!")
	count := 0
	handlerFunc := bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
		count++
		//slot := block.ToNative().(*pbcodec.Slot)
		if count > 20 {
			return fmt.Errorf("end block")
		}
		forkObj := obj.(*forkable.ForkableObject)
		zlog.Info("handling slot",
			zap.Stringer("slot", block),
			zap.Uint64("lib_num", block.LibNum),
			zap.Stringer("step", forkObj.Step),
		)
		return nil
	})
	//handlerGate := bstream.NewBlockNumGate(64803700, bstream.GateInclusive, handlerFunc)
	forkableOptions := []forkable.Option{
		forkable.WithFilters(forkable.StepsAll),
		forkable.WithLogger(zlog),
	}
	forkHandler := forkable.New(handlerFunc, forkableOptions...)
	store, err := dstore.NewDBinStore("gs://dfuseio-global-blocks-us/sol-mainnet/v1")
	require.NoError(t, err)
	source := bstream.NewFileSource(
		store,
		64803700, //64618700,
		1,
		nil,
		forkHandler,
	)
	source.Run()
	fmt.Println("end")
}
