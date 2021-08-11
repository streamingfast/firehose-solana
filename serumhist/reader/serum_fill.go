package reader

import (
	"context"
	"fmt"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/streamingfast/kvdb/store"
	"github.com/streamingfast/solana-go"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

func (m *Reader) GetFillsByTrader(ctx context.Context, trader solana.PublicKey, limit int) (fills []*pbserumhist.Fill, hasMore bool, err error) {
	prefix := keyer.EncodeFillByTraderPrefix(trader)
	zlog.Debug("get fills by trader",
		zap.Stringer("prefix", prefix),
		zap.Stringer("trader", trader),
	)
	return m.getFillsForPrefix(ctx, prefix, keyer.DecodeFillByTrader, limit)
}

func (m *Reader) GetFillsByTraderAndMarket(ctx context.Context, trader, market solana.PublicKey, limit int) (fills []*pbserumhist.Fill, hasMore bool, err error) {
	prefix := keyer.EncodeFillByTraderMarketPrefix(trader, market)
	zlog.Debug("get fills by trader and market",
		zap.Stringer("prefix", prefix),
		zap.Stringer("trader", trader),
		zap.Stringer("market", market),
	)
	return m.getFillsForPrefix(ctx, prefix, keyer.DecodeFillByTraderMarket, limit)
}

func (m *Reader) GetFillsByMarket(ctx context.Context, market solana.PublicKey, limit int) (fills []*pbserumhist.Fill, hasMore bool, err error) {
	prefix := keyer.EncodeFillByMarketPrefix(market)
	zlog.Debug("get fills by market",
		zap.Stringer("prefix", prefix),
		zap.Stringer("market", market),
	)
	return m.getFillsForMarket(ctx, prefix, limit)
}

func (m *Reader) getFillsForPrefix(ctx context.Context, prefix keyer.Prefix, decoder keyer.KeyDecoder, limit int) (out []*pbserumhist.Fill, hasMore bool, err error) {
	orderIterator := m.store.Prefix(ctx, prefix, limit+1, store.KeyOnly())
	var fillKeys [][]byte
	for orderIterator.Next() {
		if len(fillKeys) < limit {
			_, market, slotNum, trxIdx, instIdx, orderSeqNum := decoder(orderIterator.Item().Key)
			fillKeys = append(fillKeys, keyer.EncodeFill(market, slotNum, trxIdx, instIdx, orderSeqNum))
		} else {
			hasMore = true
		}
	}

	if orderIterator.Err() != nil {
		return nil, false, fmt.Errorf("unable to get fills keys: %w", err)
	}

	zlog.Debug("resolved fill keys to retrieve", zap.Int("key_count", len(fillKeys)))

	subCtx, cancelFills := context.WithCancel(ctx)
	defer cancelFills()

	fillsIter := m.store.BatchGet(subCtx, fillKeys)
	for fillsIter.Next() {
		f := &pbserumhist.Fill{}
		err := proto.Unmarshal(fillsIter.Item().Value, f)
		if err != nil {
			return nil, false, fmt.Errorf("failed to unmarshal order: %w", err)
		}

		market, slotNum, trxIdx, instIdx, orderSeqNum := keyer.DecodeFill(fillsIter.Item().Key)
		f.Market = market.String()
		f.SlotNum = slotNum
		f.TrxIdx = uint32(trxIdx)
		f.InstIdx = uint32(instIdx)
		f.OrderSeqNum = orderSeqNum
		out = append(out, f)
	}

	if fillsIter.Err() != nil {
		return nil, false, fmt.Errorf("unable to resolve fills keys: %w", err)
	}

	zlog.Debug("found fills ", zap.Int("count", len(out)), zap.Bool("has_more", hasMore))
	return
}

func (m *Reader) getFillsForMarket(ctx context.Context, prefix keyer.Prefix, limit int) (out []*pbserumhist.Fill, hasMore bool, err error) {
	zlog.Debug("get fills for market prefix",
		zap.Stringer("prefix", prefix),
	)

	fillIterator := m.store.Prefix(ctx, prefix, limit+1)
	for fillIterator.Next() {
		if len(out) < limit {
			f := &pbserumhist.Fill{}
			err := proto.Unmarshal(fillIterator.Item().Value, f)
			if err != nil {
				return nil, false, fmt.Errorf("failed to unmarshal order: %w", err)
			}

			market, slotNum, trxIdx, instIdx, orderSeqNum := keyer.DecodeFill(fillIterator.Item().Key)
			f.Market = market.String()
			f.SlotNum = slotNum
			f.TrxIdx = uint32(trxIdx)
			f.InstIdx = uint32(instIdx)
			f.OrderSeqNum = orderSeqNum
			out = append(out, f)
		} else {
			hasMore = true
		}
	}

	zlog.Debug("found fills", zap.Int("count", len(out)), zap.Bool("has_more", hasMore))
	return
}
