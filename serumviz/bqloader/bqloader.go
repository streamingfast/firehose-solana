package bqloader

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/streamingfast/dstore"
	"github.com/dfuse-io/shutter"
)

type BQLoader struct {
	*shutter.Shutter

	ctx context.Context

	client     *bigquery.Client
	dataset    *bigquery.Dataset
	store      dstore.Store
	storeUrl   string
	injector   *BigQueryInjector
	startBlock uint64

	checkpoints   map[Table]*pbserumhist.Checkpoint
	eventHandlers map[Table]*EventHandler

	traderAccountCache *tradingAccountCache
}

func New(ctx context.Context, startBlock uint64, storeUrl string, store dstore.Store, dataset *bigquery.Dataset, client *bigquery.Client) *BQLoader {
	cacheTableName := fmt.Sprintf("%s.serum.%s", dataset.ProjectID, tableTraders)

	bq := &BQLoader{
		Shutter:            shutter.New(),
		ctx:                ctx,
		client:             client,
		dataset:            dataset,
		store:              store,
		storeUrl:           storeUrl,
		injector:           NewBigQueryInjector(),
		eventHandlers:      map[Table]*EventHandler{},
		checkpoints:        map[Table]*pbserumhist.Checkpoint{},
		startBlock:         startBlock,
		traderAccountCache: newTradingAccountCache(cacheTableName, client),
	}

	bq.OnTerminating(func(err error) {
		for _, h := range bq.eventHandlers {
			h.Shutdown(err)
		}
		bq.injector.Shutdown(err)
	})

	bq.injector.OnTerminated(func(err error) {
		bq.Shutdown(err)
	})

	return bq
}

func (bq *BQLoader) Init(ctx context.Context, scratchSpaceDir string) error {
	// check that all required tables exist
	for _, table := range allTables {
		exists, err := table.Exists(ctx, bq.dataset)
		if err != nil {
			return fmt.Errorf("could not check existence of table %q: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("table %q does not exist", table)
		}

		if err := table.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize table %q: %w", table, err)
		}

	}

	if len(bq.checkpoints) == 0 {
		err := bq.readCheckpoints(ctx)
		if err != nil {
			return err
		}
	}

	var newOrderStartBlock, orderFillStartBlock, tradingAccountStartBlock = bq.startBlock, bq.startBlock, bq.startBlock // default to configured start block
	if cp, ok := bq.checkpoints[tableOrders]; ok && cp != nil {
		newOrderStartBlock = cp.LastWrittenSlotNum
	}

	if cp, ok := bq.checkpoints[tableFills]; ok && cp != nil {
		orderFillStartBlock = cp.LastWrittenSlotNum
	}

	if cp, ok := bq.checkpoints[tableTraders]; ok && cp != nil {
		tradingAccountStartBlock = cp.LastWrittenSlotNum
	}

	bq.eventHandlers[tableOrders] = NewEventHandler(newOrderStartBlock, bq.storeUrl, bq.store, bq.dataset, tableOrders, bq.injector, scratchSpaceDir)
	bq.eventHandlers[tableFills] = NewEventHandler(orderFillStartBlock, bq.storeUrl, bq.store, bq.dataset, tableFills, bq.injector, scratchSpaceDir)
	bq.eventHandlers[tableTraders] = NewEventHandler(tradingAccountStartBlock, bq.storeUrl, bq.store, bq.dataset, tableTraders, bq.injector, scratchSpaceDir)

	bq.injector.Run()

	return nil
}

func (bq *BQLoader) PrimeTradeCache(ctx context.Context) error {
	zlog.Info("priming bq trader cache")
	return bq.traderAccountCache.load(ctx)
}
