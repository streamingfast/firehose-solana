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

func (m *Manager) GetFillsByTrader(ctx context.Context, trader solana.PublicKey) ([]*pbserumhist.Fill, error) {
	prefix := keyer.EncodeOrdersPrefixByPubkey(trader)
	zlog.Debug("get fills by trader", zap.Stringer("prefix", prefix), zap.Stringer("trader", trader))
	return m.getFillsForPrefix(ctx, prefix, 100)
}

func (m *Manager) GetFillsByTraderAndMarket(ctx context.Context, trader, market solana.PublicKey) ([]*pbserumhist.Fill, error) {
	prefix := keyer.EncodeOrdersPrefixByMarketPubkey(trader, market)
	zlog.Debug("get fills by trader and market", zap.Stringer("prefix", prefix), zap.Stringer("trader", trader), zap.Stringer("market", market))
	return m.getFillsForPrefix(ctx, prefix, 100)
}

func (m *Manager) getFillsForPrefix(ctx context.Context, prefix keyer.Prefix, limit int) ([]*pbserumhist.Fill, error) {
	zlog.Debug("get fills for prefix", zap.Stringer("prefix", prefix))
	orderIterator := m.store.Prefix(ctx, prefix, limit)

	var fillKeys [][]byte
	for orderIterator.Next() {
		k := orderIterator.Item().Key
		_, market, orderSeqNum, slotNum := keyer.DecodeOrdersByPubkey(k)
		fk := keyer.EncodeFillData(market, orderSeqNum, slotNum)
		if traceEnabled {
			zlog.Debug("order key", zap.Stringer("key", fk), zap.Stringer("market", market), zap.Uint64("order_seq_num", orderSeqNum), zap.Uint64("slot_num", slotNum), zap.Stringer("fill_key", keyer.Key(k)))
		}
		fillKeys = append(fillKeys, fk)
	}

	if err := orderIterator.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate orders: %w", err)
	}

	zlog.Debug("found fills keys", zap.Int("count", len(fillKeys)))
	getFillsCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	fillsIterator := m.store.BatchGet(getFillsCtx, fillKeys)

	var fills []*pbserumhist.Fill
	for fillsIterator.Next() {
		f := &pbserumhist.Fill{}
		err := proto.Unmarshal(orderIterator.Item().Value, f)
		if err != nil {
			fillsIterator.PushFinished()
			return nil, fmt.Errorf("failed to unmarshal order: %w", err)
		}

		fills = append(fills, f)
	}

	if err := orderIterator.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate fills: %w", err)
	}
	zlog.Debug("found fills ", zap.Int("count", len(fills)))

	return fills, nil
}
