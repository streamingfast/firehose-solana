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

const (
	newOrder       = "orders"
	fillOrder      = "fills"
	tradingAccount = "traders"
	markets        = "markets"
	tokens         = "tokens"

	processedFiles = "processedFiles"
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
	cacheTableName := fmt.Sprintf("%s.serum.%s", dataset.ProjectID, tradingAccount)
	checkpointsTableName := fmt.Sprintf("%s.serum.%s", dataset.ProjectID, processedFiles)
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
	if cp, ok := bq.checkpoints[newOrder]; ok && cp != nil {
		newOrderStartBlock = cp.LastWrittenSlotNum
	}

	if cp, ok := bq.checkpoints[fillOrder]; ok && cp != nil {
		orderFillStartBlock = cp.LastWrittenSlotNum
	}

	if cp, ok := bq.checkpoints[tradingAccount]; ok && cp != nil {
		tradingAccountStartBlock = cp.LastWrittenSlotNum
	}

	bq.eventHandlers[newOrder] = NewEventHandler(newOrderStartBlock, bq.storeUrl, bq.store, newOrder, bq.loader, path.Join(scratchSpaceDir, newOrder))
	bq.eventHandlers[fillOrder] = NewEventHandler(orderFillStartBlock, bq.storeUrl, bq.store, fillOrder, bq.loader, path.Join(scratchSpaceDir, fillOrder))
	bq.eventHandlers[tradingAccount] = NewEventHandler(tradingAccountStartBlock, bq.storeUrl, bq.store, tradingAccount, bq.loader, path.Join(scratchSpaceDir, tradingAccount))

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
