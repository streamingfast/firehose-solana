package serumhist

import (
	"context"
	"fmt"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type Manager struct {
	store store.KVStore
}

func NewManager(store store.KVStore) *Manager {
	return &Manager{
		store: store,
	}
}

func (m *Manager) GetFillsByTrader(ctx context.Context, trader solana.PublicKey, limit int) (fills []*pbserumhist.Fill, hasMore bool, err error) {
	prefix := keyer.EncodeFillByTraderPrefix(trader)
	zlog.Debug("get fills by trader",
		zap.Stringer("prefix", prefix),
		zap.Stringer("trader", trader),
	)
	return m.getFillsForPrefix(ctx, prefix, keyer.DecodeFillByTrader, limit)
}

func (m *Manager) GetFillsByMarket(ctx context.Context, market solana.PublicKey, limit int) (fills []*pbserumhist.Fill, hasMore bool, err error) {
	prefix := keyer.EncodeFillByMarketPrefix(market)
	zlog.Debug("get fills by trader",
		zap.Stringer("prefix", prefix),
		zap.Stringer("market", market),
	)
	return m.getFillsForPrefix(ctx, prefix, keyer.DecodeFillByTrader, limit)
}

func (m *Manager) GetFillsByTraderAndMarket(ctx context.Context, trader, market solana.PublicKey, limit int) (fills []*pbserumhist.Fill, hasMore bool, err error) {
	prefix := keyer.EncodeFillByTraderMarketPrefix(trader, market)
	zlog.Debug("get fills by trader and market",
		zap.Stringer("prefix", prefix),
		zap.Stringer("trader", trader),
		zap.Stringer("market", market),
	)
	return m.getFillsForPrefix(ctx, prefix, keyer.DecodeFillByMarketTrader, limit)
}

func (m *Manager) getFillsForPrefix(ctx context.Context, prefix keyer.Prefix, decoder keyer.KeyDecoder, limit int) (out []*pbserumhist.Fill, hasMore bool, err error) {
	zlog.Debug("get fills for prefix",
		zap.Stringer("prefix", prefix),
	)
	orderIterator := m.store.Prefix(ctx, prefix, limit+1)

	for orderIterator.Next() {
		f := &pbserumhist.Fill{}
		err := proto.Unmarshal(orderIterator.Item().Value, f)
		if err != nil {
			return nil, false, fmt.Errorf("failed to unmarshal order: %w", err)
		}

		trader, market, slotNum, trxIdx, instIdx, orderSeqNum := decoder(orderIterator.Item().Key)
		f.Trader = trader.String()
		f.Market = market.String()
		f.SlotNum = slotNum
		f.TrxIdx = uint32(trxIdx)
		f.InstIdx = uint32(instIdx)
		f.OrderSeqNum = orderSeqNum

		if len(out) < limit {
			out = append(out, f)
		} else {
			hasMore = true
		}
	}
	if err := orderIterator.Err(); err != nil {
		return nil, false, fmt.Errorf("failed to iterate fills: %w", err)
	}
	zlog.Debug("found fills ", zap.Int("count", len(out)), zap.Bool("has_more", hasMore))

	return
}
