package bqloader

import (
	"context"
	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/dfuse-io/dstore"
	"github.com/linkedin/goavro/v2"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type avroAggregator struct {
	scratchFilename string
	count, batch    int
	ocfFile         *os.File
	ocfWriter       *goavro.OCFWriter
	t0              time.Time

	scratchSpaceDir string
	lock            sync.Mutex
	flushInterval   time.Duration

	store dstore.Store
}

// NewAvroAggregator creates a new Avro event aggregator. The `scratchSpaceDir` is expected to be a local file system path.
func NewAvroAggregator(ctx context.Context, scratchSpaceDir string, store dstore.Store, flushInterval time.Duration) *avroAggregator {
	agg := &avroAggregator{
		scratchSpaceDir: scratchSpaceDir,
		scratchFilename: filepath.Join(scratchSpaceDir, "pending.ocf"),
		store:           store,
		flushInterval:   flushInterval,
	}

	return agg
}

func (agg *avroAggregator) handleOrderCreatedEvent(e *serumhist.NewOrder) error { return nil }
func (agg *avroAggregator) handleOrderFilledEvent(e *serumhist.NewOrder) error { return nil }
func (agg *avroAggregator) handleTraderAccount(e *serumhist.NewOrder) error { return nil }

func (agg *avroAggregator) flushFile(ctx context.Context) error {
	return nil
}

func OrderCreatedEventToAvro(e serumhist.NewOrder) map[string]interface{} {
	return nil
}

func OrderFilledEventToAvro(e serumhist.NewOrder) map[string]interface{} {
	return nil
}

func TraderAccountToAvro() map[string]interface{} {
	return nil
}