package bqloader

import (
	"context"
	"fmt"
	"path"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"

	"cloud.google.com/go/bigquery"
	"github.com/dfuse-io/dfuse-solana/registry"
	"github.com/dfuse-io/dstore"
)

type BQLoader struct {
	ctx context.Context

	client         *bigquery.Client
	dataset        *bigquery.Dataset
	store          dstore.Store
	storeUrl       string
	registryServer *registry.Server
	loader         *BigQueryLoader
	startBlock     uint64

	checkpoints map[string]*pbserumhist.Checkpoint

	traderAccountCache *tradingAccountCache
	eventHandlers      map[string]*EventHandler
}

func New(ctx context.Context, startBlock uint64, storeUrl string, store dstore.Store, dataset *bigquery.Dataset, client *bigquery.Client, registry *registry.Server) *BQLoader {
	cacheTableName := fmt.Sprintf("%s.serum.%s", dataset.ProjectID, tableTraders)
	checkpointsTableName := fmt.Sprintf("%s.serum.%s", dataset.ProjectID, tableProcessedFiles)
	bq := &BQLoader{
		ctx:                ctx,
		client:             client,
		dataset:            dataset,
		store:              store,
		storeUrl:           storeUrl,
		registryServer:     registry,
		loader:             NewBigQueryLoader(dataset, client, checkpointsTableName),
		eventHandlers:      map[string]*EventHandler{},
		startBlock:         startBlock,
		checkpoints:        map[string]*pbserumhist.Checkpoint{},
		traderAccountCache: newTradingAccountCache(cacheTableName, client),
	}

	return bq
}

func (bq *BQLoader) InitHandlers(ctx context.Context, scratchSpaceDir string) error {
	if len(bq.checkpoints) == 0 {
		err := bq.setCheckpoints(ctx)
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

	bq.eventHandlers[tableOrders] = NewEventHandler(newOrderStartBlock, bq.storeUrl, bq.store, tableOrders, bq.loader, path.Join(scratchSpaceDir, tableOrders))
	bq.eventHandlers[tableFills] = NewEventHandler(orderFillStartBlock, bq.storeUrl, bq.store, tableFills, bq.loader, path.Join(scratchSpaceDir, tableFills))
	bq.eventHandlers[tableTraders] = NewEventHandler(tradingAccountStartBlock, bq.storeUrl, bq.store, tableTraders, bq.loader, path.Join(scratchSpaceDir, tableTraders))

	bq.loader.Run(ctx)

	return nil
}

func (bq *BQLoader) PrimeTradeCache(ctx context.Context) error {
	zlog.Info("priming bq trader cache")
	return bq.traderAccountCache.load(ctx)
}

func (bq *BQLoader) Close() error {
	for _, h := range bq.eventHandlers {
		h.Shutdown(nil)
	}

	bq.loader.Shutdown(nil)
	return nil
}
