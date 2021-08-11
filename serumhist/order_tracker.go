package serumhist

import (
	"context"
	"fmt"
	"sync"

	"github.com/dfuse-io/logging"
	"github.com/streamingfast/solana-go"
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

func (m *OrderManager) emit(event interface{}) {
	m.subscriptionsLock.RLock()
	defer m.subscriptionsLock.RUnlock()
	for _, sub := range m.subscriptions {
		//if sub.orderNum == event.GetEventRef().OrderSeqNum && sub.market.String() == event.GetEventRef().Market.String() {
		sub.Push(event)
		//}
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

	logger.Debug("unsubscribed", zap.Int("new_length", len(m.subscriptions)))
	return true
}

type subscription struct {
	traceID  string
	orderNum uint64
	market   solana.PublicKey
	// TODO: unify with an interface
	// the event is of type  orderFillEvent || orderExecutedEvent || orderClosedEvent || orderCancelledEvent
	conn     chan interface{}
	closed   bool
	quitOnce sync.Once
	sync.Mutex
}

func newSubscription(chanSize int, orderNum uint64, market solana.PublicKey) (out *subscription, err error) {
	s := &subscription{
		conn:     make(chan interface{}, chanSize),
		orderNum: orderNum,
		market:   market,
	}

	return s, nil
}

func (s *subscription) Push(event interface{}) {
	if s.closed {
		return
	}

	if len(s.conn) == cap(s.conn) {
		s.quitOnce.Do(func() {
			zlog.Info("reach max buffer size for stream, closing channel", zap.String("subscription_trace_id", s.traceID))
			close(s.conn)
			s.closed = true
		})
		return
	}

	s.conn <- event
}
