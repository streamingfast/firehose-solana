package serumhist

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
	"sync"

	"github.com/dfuse-io/dfuse-solana/serumhist/metrics"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/logging"
	"github.com/dfuse-io/solana-go"
	"go.uber.org/zap"
)

type StatefulOrder struct {
	order     *pbserumhist.Order
	cancelled *pbserumhist.InstructionRef
	state     pbserumhist.OrderTransition_State
}

func newStatefulOrder() *StatefulOrder {
	return &StatefulOrder{
		order: &pbserumhist.Order{},
		state: pbserumhist.OrderTransition_STATE_UNKNOWN,
	}
}

// TODO: unify with an interface
// the event is of type  orderFillEvent || orderExecutedEvent || orderClosedEvent || orderCancelledEvent
func (s *StatefulOrder) applyEvent(event interface{}) (*pbserumhist.OrderTransition, error) {
	out := &pbserumhist.OrderTransition{
		PreviousState: s.state,
	}

	switch v := event.(type) {
	case *orderNewEvent:
		zlog.Debug("applying new order event")
		s.state = pbserumhist.OrderTransition_STATE_APPROVED
		out.Transition = pbserumhist.OrderTransition_TRANS_ACCEPTED

		s.order.Market = v.orderEventRef.market.String()
		s.order.SlotNum = v.orderEventRef.slotNumber
		s.order.TrxIdx = v.orderEventRef.trxIdx
		s.order.InstIdx = v.orderEventRef.instIdx
	case *orderFillEvent:
		zlog.Debug("applying fill order event")
		s.state = pbserumhist.OrderTransition_STATE_PARTIAL
		out.Transition = pbserumhist.OrderTransition_TRANS_FILLED

		fill := v.fill
		fill.Market = v.orderEventRef.market.String()
		fill.SlotNum = v.orderEventRef.slotNumber
		fill.TrxIdx = v.orderEventRef.trxIdx
		fill.InstIdx = v.orderEventRef.instIdx
		fill.OrderSeqNum = v.orderEventRef.orderSeqNum
		s.order.Fills = append(s.order.Fills, fill)
		out.AddedFill = fill
	case *orderExecutedEvent:
		zlog.Debug("applying executed event")
		s.state = pbserumhist.OrderTransition_STATE_EXECUTED
		out.Transition = pbserumhist.OrderTransition_TRANS_EXECUTED
	case *orderCancelledEvent:
		zlog.Debug("applying cancellation order event")
		s.state = pbserumhist.OrderTransition_STATE_CANCELLED
		out.Transition = pbserumhist.OrderTransition_TRANS_CANCELLED

		instrRef := v.instrRef
		instrRef.SlotNum = v.orderEventRef.slotNumber
		instrRef.TrxIdx = v.orderEventRef.trxIdx
		instrRef.InstIdx = v.orderEventRef.instIdx

		s.cancelled = instrRef
	case *orderClosedEvent:
		if len(s.order.Fills) == 0 {
			zlog.Debug("applying closed order event as a cancellation")
			s.state = pbserumhist.OrderTransition_STATE_CANCELLED
			out.Transition = pbserumhist.OrderTransition_TRANS_CANCELLED

			instrRef := v.instrRef
			instrRef.SlotNum = v.orderEventRef.slotNumber
			instrRef.TrxIdx = v.orderEventRef.trxIdx
			instrRef.InstIdx = v.orderEventRef.instIdx
			s.cancelled = instrRef
		} else {
			zlog.Debug("applying closed order event as an executed")
			s.state = pbserumhist.OrderTransition_STATE_EXECUTED
			out.Transition = pbserumhist.OrderTransition_TRANS_EXECUTED
		}
	}

	out.CurrentState = s.state
	out.Order = s.order
	out.Cancellation = s.cancelled
	return out, nil
}

func GetInitializeOrder(ctx context.Context, kvdb store.KVStore, market solana.PublicKey, orderNum uint64) (*StatefulOrder, *pbserumhist.OrderTransition, error) {
	statefulOrder := newStatefulOrder()
	orderKeyPrefix := keyer.EncodeOrderPrefix(market, orderNum)

	zlog.Debug("get order",
		zap.Stringer("prefix", orderKeyPrefix),
	)
	itr := kvdb.Prefix(ctx, orderKeyPrefix, 0)

	seenOrderKey := false
	transition := &pbserumhist.OrderTransition{
		PreviousState: pbserumhist.OrderTransition_STATE_UNKNOWN,
		CurrentState:  pbserumhist.OrderTransition_STATE_UNKNOWN,
	}
	var err error
	for itr.Next() {
		seenOrderKey = true
		var event interface{}
		eventByte, market, slotNum, trxIdx, instIdx, orderSeqNum := keyer.DecodeOrder(itr.Item().Key)
		switch eventByte {
		case keyer.OrderEventTypeNew:
			order := &pbserumhist.Order{}
			err := proto.Unmarshal(itr.Item().Value, order)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal order: %w", err)
			}
			event = &orderNewEvent{
				orderEventRef: orderEventRef{
					market:      market,
					orderSeqNum: orderSeqNum,
					slotNumber:  slotNum,
					trxIdx:      uint32(trxIdx),
					instIdx:     uint32(instIdx),
				},
				order: order,
			}
		case keyer.OrderEventTypeFill:
			fill := &pbserumhist.Fill{}
			err := proto.Unmarshal(itr.Item().Value, fill)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal fil: %w", err)
			}
			event = &orderFillEvent{
				orderEventRef: orderEventRef{
					market:      market,
					orderSeqNum: orderSeqNum,
					slotNumber:  slotNum,
					trxIdx:      uint32(trxIdx),
					instIdx:     uint32(instIdx),
				},
				fill: fill,
			}
		case keyer.OrderEventTypeExecuted:
			event = &orderExecutedEvent{
				orderEventRef: orderEventRef{
					market:      market,
					orderSeqNum: orderSeqNum,
					slotNumber:  slotNum,
					trxIdx:      uint32(trxIdx),
					instIdx:     uint32(instIdx),
				},
			}
		case keyer.OrderEventTypeCancel:
			instrRef := &pbserumhist.InstructionRef{}
			err := proto.Unmarshal(itr.Item().Value, instrRef)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal instruction ref: %w", err)
			}
			event = &orderCancelledEvent{
				orderEventRef: orderEventRef{
					market:      market,
					orderSeqNum: orderSeqNum,
					slotNumber:  slotNum,
					trxIdx:      uint32(trxIdx),
					instIdx:     uint32(instIdx),
				},
				instrRef: instrRef,
			}
		case keyer.OrderEventTypeClose:
			instrRef := &pbserumhist.InstructionRef{}
			err := proto.Unmarshal(itr.Item().Value, instrRef)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal instruction ref: %w", err)
			}
			event = &orderClosedEvent{
				orderEventRef: orderEventRef{
					market:      market,
					orderSeqNum: orderSeqNum,
					slotNumber:  slotNum,
					trxIdx:      uint32(trxIdx),
					instIdx:     uint32(instIdx),
				},
				instrRef: instrRef,
			}
		}
		if transition, err = statefulOrder.applyEvent(event); err != nil {
			return nil, nil, fmt.Errorf("error applying the event on the stateful order event type 0x%s: %w", hex.EncodeToString([]byte{eventByte}), err)
		}
	}
	// override the transition on the initlization stage
	transition.Transition = pbserumhist.OrderTransition_TRANS_INIT
	if !seenOrderKey {
		zlog.Info("unable to initialize order. no keys found",
			zap.Stringer("market", market),
			zap.Uint64("order_num", orderNum),
		)
		return statefulOrder, transition, nil

	}

	return statefulOrder, transition, nil
}

type OrderManager struct {
	subscriptionsLock sync.RWMutex
	subscriptions     []*subscription
}

func newOrderManager() *OrderManager {
	return &OrderManager{
		subscriptions: []*subscription{},
	}
}

func (m *OrderManager) emit(event interface{}, orderNum uint64, market solana.PublicKey) {
	m.subscriptionsLock.RLock()
	defer m.subscriptionsLock.RUnlock()
	for _, sub := range m.subscriptions {
		if sub.orderNum == orderNum && sub.market.String() == market.String() {
			sub.Push(event)
		}
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
