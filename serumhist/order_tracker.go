package serumhist

import (
	"context"
	"fmt"
	"sync"

	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/solana-go"
	"go.uber.org/zap"
)

type OrderManager struct {
	subscriptionsLock sync.RWMutex
	subscriptions     []*subscription
}

func newOrderManager() *OrderManager {
	return &OrderManager{
		subscriptions: []*subscription{},
	}
}

func (m *OrderManager) subscribe(orderNum uint64, market solana.PublicKey, logger *zap.Logger) (*subscription, error) {
	chanSize := 2000 // TODO fix me maybe
	logger.Debug("creating new subscription",
		zap.Int("channel_size", chanSize),
		zap.Uint64("order_num", orderNum),
		zap.Stringer("market", market),
	)

	sub, err := newSubscription(chanSize, orderNum, market)
	if err != nil {
		return nil, fmt.Errorf("unable to create new subscription: %w", err)
	}

	m.subscriptionsLock.Lock()
	defer m.subscriptionsLock.Unlock()

	m.subscriptions = append(m.subscriptions, sub)
	metrics.ActiveOrderTrackingSubscription.Inc()

	logger.Debug("subscribed",
		zap.Int("new_length", len(m.subscriptions)),
		zap.Uint64("order_num", orderNum),
		zap.Stringer("market", market),
	)
	return sub, nil
}

func (m *OrderManager) unsubscribe(ctx context.Context, toRemove *subscription) bool {
	logger := logging.Logger(ctx, zlog)

	m.subscriptionsLock.Lock()
	defer m.subscriptionsLock.Unlock()

	var newListeners []*subscription
	for _, sub := range m.subscriptions {
		if sub != toRemove {
			newListeners = append(newListeners, sub)
		}
	}
	m.subscriptions = newListeners
	metrics.ActiveOrderTrackingSubscription.Dec()

	logger.Debug("unsubscribed", zap.Int("new_length", len(m.subscriptions)))
	return true
}

type subscription struct {
	traceID  string
	orderNum uint64
	market   solana.PublicKey
	conn     chan *pbserumhist.OrderTransition
	closed   bool
	quitOnce sync.Once
	sync.Mutex
}

func newSubscription(chanSize int, orderNum uint64, market solana.PublicKey) (out *subscription, err error) {
	s := &subscription{
		conn:     make(chan *pbserumhist.OrderTransition, chanSize),
		orderNum: orderNum,
		market:   market,
	}

	return s, nil
}
