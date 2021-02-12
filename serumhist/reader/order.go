package reader

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/solana-go"
	"go.uber.org/zap"
)

func (r *Reader) GetInitializeOrder(ctx context.Context, market solana.PublicKey, orderNum uint64) (*pbserumhist.OrderTransition, error) {
	out := &pbserumhist.OrderTransition{
		PreviousState: pbserumhist.OrderTransition_STATE_UNKNOWN,
		Transition:    pbserumhist.OrderTransition_TRANS_INIT,
		Order:                &pbserumhist.Order{},
		AddedFill:            nil,
	}
	orderKeyPrefix := keyer.EncodeOrderPrefix(market, orderNum)

	zlog.Debug("get order",
		zap.Stringer("prefix", orderKeyPrefix),
	)
	itr := r.store.Prefix(ctx, orderKeyPrefix, 0)
	var fillKeys [][]byte
	seenOrderKey := false
	for itr.Next() {
		seenOrderKey = true
		event , market , slotNum, trxIdx, instIdx, orderSeqNum  := keyer.DecodeOrder(itr.Item().Key)
		switch event {
		case keyer.OrderEventTypeNew:
			err := proto.Unmarshal(itr.Item().Value, out.Order)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal order: %w", err)
			}
			out.Order.Market = market.String()
			out.Order.SlotNum = slotNum
			out.Order.TrxIdx = uint32(trxIdx)
			out.Order.InstIdx = uint32(instIdx)
			out.CurrentState = pbserumhist.OrderTransition_STATE_APPROVED
		case keyer.OrderEventTypeFill:
			fillKeys = append(fillKeys, keyer.EncodeFill(market, slotNum, trxIdx, instIdx, orderSeqNum))
			out.CurrentState = pbserumhist.OrderTransition_STATE_PARTIAL
		case keyer.OrderEventTypeCancel:
			out.Cancellation = &pbserumhist.InstructionRef{}
			err := proto.Unmarshal(itr.Item().Value, out.Cancellation)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal order: %w", err)
			}
			out.Cancellation.SlotNum = slotNum
			out.Cancellation.TrxIdx = uint32(trxIdx)
			out.Cancellation.InstIdx = uint32(instIdx)
			out.CurrentState = pbserumhist.OrderTransition_STATE_CANCELLED
		case keyer.OrderEventTypeExecuted:
			out.CurrentState = pbserumhist.OrderTransition_STATE_EXECUTED
		case keyer.OrderEventTypeClose:
			// since the keys are sorted alphanemurically, we should only get
			// OrderEventTypeClose after receiving all Fill
			if len(fillKeys) == 0 {
				out.Cancellation = &pbserumhist.InstructionRef{}
				// since there no fill we can assume the order was canceled
				err := proto.Unmarshal(itr.Item().Value, out.Cancellation)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal order: %w", err)
				}
				out.Cancellation.SlotNum = slotNum
				out.Cancellation.TrxIdx = uint32(trxIdx)
				out.Cancellation.InstIdx = uint32(instIdx)
				out.CurrentState = pbserumhist.OrderTransition_STATE_CANCELLED
			} else {
				// since there are fill we can assume the order was executed
				out.CurrentState = pbserumhist.OrderTransition_STATE_EXECUTED
			}

		}
	}

	if !seenOrderKey {
		zlog.Info("unable to initialize order. no keys found",
			zap.Stringer("market", market),
			zap.Uint64("order_num", orderNum),
		)
		return nil, nil
	}

	zlog.Debug("stitched a serum order",
		zap.Stringer("previous_state", out.PreviousState),
		zap.Stringer("current_state", out.CurrentState),
		zap.Stringer("transition", out.Transition),
		zap.Int("fill_count", len(fillKeys)),
	)

	fillIter := r.store.BatchGet(ctx, fillKeys)
	for fillIter.Next() {
		f := &pbserumhist.Fill{}
		err := proto.Unmarshal(fillIter.Item().Value, f)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal order: %w", err)
		}

		market, slotNum, trxIdx, instIdx, orderSeqNum := keyer.DecodeFill(fillIter.Item().Key)
		f.Market = market.String()
		f.SlotNum = slotNum
		f.TrxIdx = uint32(trxIdx)
		f.InstIdx = uint32(instIdx)
		f.OrderSeqNum = orderSeqNum
		out.Order.Fills = append(out.Order.Fills, f)
	}
	zlog.Debug("serum order transition retrieved")
	return out, nil
}
