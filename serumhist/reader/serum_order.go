package reader

import (
	"context"
	"encoding/hex"
	"fmt"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/solana-go"
	"github.com/golang/protobuf/proto"
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

func (s *StatefulOrder) applyEvent(e interface{}) (*pbserumhist.OrderTransition, error) {
	out := &pbserumhist.OrderTransition{
		PreviousState: s.state,
	}

	switch v := e.(type) {
	case *serumhist.NewOrder:
		zlog.Debug("applying new order event")
		s.state = pbserumhist.OrderTransition_STATE_APPROVED
		out.Transition = pbserumhist.OrderTransition_TRANS_ACCEPTED
		s.order = v.Order
		s.order.Market = v.Ref.Market.String()
		s.order.SlotNum = v.Ref.SlotNumber
		s.order.TrxIdx = v.Ref.TrxIdx
		s.order.InstIdx = v.Ref.InstIdx
	case *serumhist.FillEvent:
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
	case *serumhist.OrderExecuted:
		zlog.Debug("applying executed event")
		s.state = pbserumhist.OrderTransition_STATE_EXECUTED
		out.Transition = pbserumhist.OrderTransition_TRANS_EXECUTED

	case *serumhist.OrderCancelled:
		zlog.Debug("applying cancellation order event")
		s.state = pbserumhist.OrderTransition_STATE_CANCELLED
		out.Transition = pbserumhist.OrderTransition_TRANS_CANCELLED

		instrRef := v.InstrRef
		instrRef.SlotNum = v.Ref.SlotNumber
		instrRef.TrxIdx = v.Ref.TrxIdx
		instrRef.InstIdx = v.Ref.InstIdx

		s.cancelled = instrRef
	case *serumhist.OrderClosed:
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

func (m *Reader) GetOrder(ctx context.Context, market solana.PublicKey, orderNum uint64) (*pbserumhist.Order, error) {
	statefulOrder := newStatefulOrder()
	orderKeyPrefix := keyer.EncodeOrderPrefix(market, orderNum)

	zlog.Debug("get order",
		zap.Stringer("prefix", orderKeyPrefix),
	)
	itr := m.store.Prefix(ctx, orderKeyPrefix, 0)

	seenOrderKey := false
	transition := &pbserumhist.OrderTransition{
		PreviousState: pbserumhist.OrderTransition_STATE_UNKNOWN,
		CurrentState:  pbserumhist.OrderTransition_STATE_UNKNOWN,
	}
	var err error
	for itr.Next() {
		seenOrderKey = true
		var e interface{}
		eventByte, market, slotNum, trxIdx, instIdx, orderSeqNum := keyer.DecodeOrder(itr.Item().Key)
		zlog.Debug("found new order key prefix",
			zap.String("event", hex.EncodeToString([]byte{eventByte})),
			zap.Stringer("market", market),
			zap.Uint64("slot_num", slotNum),
			zap.Uint64("trx_index", trxIdx),
			zap.Uint64("inst_index", instIdx),
			zap.Uint64("orde_seq_num", orderSeqNum),
		)

		switch eventByte {
		case keyer.OrderEventTypeNew:
			order := &pbserumhist.Order{}
			err := proto.Unmarshal(itr.Item().Value, order)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal order: %w", err)
			}

			e = &serumhist.NewOrder{
				Ref: serumhist.Ref{
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
			fillKey := keyer.EncodeFill(market, slotNum, trxIdx, instIdx, orderSeqNum)
			v, err := m.store.Get(ctx, fillKey)
			if err != nil {
				zlog.Warn("unable to find fills for order",
					zap.Stringer("market", market),
					zap.Uint64("slot_num", slotNum),
					zap.Uint64("trx_idx", trxIdx),
					zap.Uint64("inst_indx", instIdx),
					zap.Uint64("order_seq_num", orderSeqNum),
				)
				continue
			}
			err = proto.Unmarshal(v, fill)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal fil: %w", err)
			}
			e = &serumhist.FillEvent{
				Ref: serumhist.Ref{
					Market:      market,
					OrderSeqNum: orderSeqNum,
					SlotNumber:  slotNum,
					TrxIdx:      uint32(trxIdx),
					InstIdx:     uint32(instIdx),
				},
				Fill: fill,
			}
		case keyer.OrderEventTypeExecuted:
			e = &serumhist.OrderExecuted{
				Ref: serumhist.Ref{
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
				return nil, fmt.Errorf("failed to unmarshal instruction ref: %w", err)
			}
			e = &serumhist.OrderCancelled{
				Ref: serumhist.Ref{
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
				return nil, fmt.Errorf("failed to unmarshal instruction ref: %w", err)
			}
			e = &serumhist.OrderClosed{
				Ref: serumhist.Ref{
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
			return nil, fmt.Errorf("error applying the event on the stateful order event type 0x%s: %w", hex.EncodeToString([]byte{eventByte}), err)
		}
	}
	if !seenOrderKey {
		zlog.Info("unable to initialize order. no keys found",
			zap.Stringer("market", market),
			zap.Uint64("order_num", orderNum),
		)
		return nil, fmt.Errorf("Order not found %d", orderNum)
	}

	return transition.Order, nil
}
