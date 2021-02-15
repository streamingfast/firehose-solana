package serumhist

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/dfuse-io/dfuse-solana/serumhist/event"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"

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
func (s *StatefulOrder) applyEvent(e event.Eventeable) (*pbserumhist.OrderTransition, error) {
	out := &pbserumhist.OrderTransition{
		PreviousState: s.state,
	}

	switch v := e.(type) {
	case *event.NewOrder:
		zlog.Debug("applying new order event")
		s.state = pbserumhist.OrderTransition_STATE_APPROVED
		out.Transition = pbserumhist.OrderTransition_TRANS_ACCEPTED

		s.order.Market = v.Ref.Market.String()
		s.order.SlotNum = v.Ref.SlotNumber
		s.order.TrxIdx = v.Ref.TrxIdx
		s.order.InstIdx = v.Ref.InstIdx
	case *event.Fill:
		zlog.Debug("applying fill order event")
		s.state = pbserumhist.OrderTransition_STATE_PARTIAL
		out.Transition = pbserumhist.OrderTransition_TRANS_FILLED

		fill := v.Fill
		fill.Market = v.Ref.Market.String()
		fill.SlotNum = v.Ref.SlotNumber
		fill.TrxIdx = v.Ref.TrxIdx
		fill.InstIdx = v.Ref.InstIdx
		fill.OrderSeqNum = v.Ref.OrderSeqNum
		s.order.Fills = append(s.order.Fills, fill)
		out.AddedFill = fill
	case *event.OrderExecuted:
		zlog.Debug("applying executed event")
		s.state = pbserumhist.OrderTransition_STATE_EXECUTED
		out.Transition = pbserumhist.OrderTransition_TRANS_EXECUTED
	case *event.OrderCancelled:
		zlog.Debug("applying cancellation order event")
		s.state = pbserumhist.OrderTransition_STATE_CANCELLED
		out.Transition = pbserumhist.OrderTransition_TRANS_CANCELLED

		instrRef := v.InstrRef
		instrRef.SlotNum = v.Ref.SlotNumber
		instrRef.TrxIdx = v.Ref.TrxIdx
		instrRef.InstIdx = v.Ref.InstIdx

		s.cancelled = instrRef
	case *event.OrderClosed:
		if len(s.order.Fills) == 0 {
			zlog.Debug("applying closed order event as a cancellation")
			s.state = pbserumhist.OrderTransition_STATE_CANCELLED
			out.Transition = pbserumhist.OrderTransition_TRANS_CANCELLED

			instrRef := v.InstrRef
			instrRef.SlotNum = v.Ref.SlotNumber
			instrRef.TrxIdx = v.Ref.TrxIdx
			instrRef.InstIdx = v.Ref.InstIdx
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
		var e event.Eventeable
		eventByte, market, slotNum, trxIdx, instIdx, orderSeqNum := keyer.DecodeOrder(itr.Item().Key)
		switch eventByte {
		case keyer.OrderEventTypeNew:
			order := &pbserumhist.Order{}
			err := proto.Unmarshal(itr.Item().Value, order)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal order: %w", err)
			}
			e = &event.NewOrder{
				Ref: &event.Ref{
					Market:      market,
					OrderSeqNum: orderSeqNum,
					SlotNumber:  slotNum,
					TrxIdx:      uint32(trxIdx),
					InstIdx:     uint32(instIdx),
				},
				Order: order,
			}
		case keyer.OrderEventTypeFill:
			fill := &pbserumhist.Fill{}
			err := proto.Unmarshal(itr.Item().Value, fill)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal fil: %w", err)
			}
			e = &event.Fill{
				Ref: &event.Ref{
					Market:      market,
					OrderSeqNum: orderSeqNum,
					SlotNumber:  slotNum,
					TrxIdx:      uint32(trxIdx),
					InstIdx:     uint32(instIdx),
				},
				Fill: fill,
			}
		case keyer.OrderEventTypeExecuted:
			e = &event.OrderExecuted{
				Ref: &event.Ref{
					Market:      market,
					OrderSeqNum: orderSeqNum,
					SlotNumber:  slotNum,
					TrxIdx:      uint32(trxIdx),
					InstIdx:     uint32(instIdx),
				},
			}
		case keyer.OrderEventTypeCancel:
			instrRef := &pbserumhist.InstructionRef{}
			err := proto.Unmarshal(itr.Item().Value, instrRef)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal instruction ref: %w", err)
			}
			e = &event.OrderCancelled{
				Ref: &event.Ref{
					Market:      market,
					OrderSeqNum: orderSeqNum,
					SlotNumber:  slotNum,
					TrxIdx:      uint32(trxIdx),
					InstIdx:     uint32(instIdx),
				},
				InstrRef: instrRef,
			}
		case keyer.OrderEventTypeClose:
			instrRef := &pbserumhist.InstructionRef{}
			err := proto.Unmarshal(itr.Item().Value, instrRef)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal instruction ref: %w", err)
			}
			e = &event.OrderClosed{
				Ref: &event.Ref{
					Market:      market,
					OrderSeqNum: orderSeqNum,
					SlotNumber:  slotNum,
					TrxIdx:      uint32(trxIdx),
					InstIdx:     uint32(instIdx),
				},
				InstrRef: instrRef,
			}
		}
		if transition, err = statefulOrder.applyEvent(e); err != nil {
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

func (m *OrderManager) emit(event event.Eventeable) {
	m.subscriptionsLock.RLock()
	defer m.subscriptionsLock.RUnlock()
	for _, sub := range m.subscriptions {
		if sub.orderNum == event.GetEventRef().OrderSeqNum && sub.market.String() == event.GetEventRef().Market.String() {
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
