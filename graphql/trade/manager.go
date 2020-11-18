package trade

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/dfuse-io/dfuse-solana/transaction"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/serum"
	"go.uber.org/zap"
)

type instructionWrapper struct {
	Decoded      *serum.Instruction
	Compiled     *solana.CompiledInstruction
	TrxSignature string
	TrxError     interface{}
}

type Subscription struct {
	Stream                  chan *instructionWrapper
	account                 solana.PublicKey
	backfillCompleted       *atomic.Bool
	backfilledTrxSignatures map[string]bool
	pushLock                sync.Mutex
	toSendLiveInstructions  []*instructionWrapper
	Err                     error
}

func NewSubscription(account solana.PublicKey) *Subscription {
	return &Subscription{
		account:                 account,
		Stream:                  make(chan *instructionWrapper, 200),
		backfillCompleted:       atomic.NewBool(false),
		backfilledTrxSignatures: map[string]bool{},
		toSendLiveInstructions:  []*instructionWrapper{},
	}
}

func (s *Subscription) pushSafe(backfilling bool, inst *instructionWrapper) {
	s.pushLock.Lock()
	defer s.pushLock.Unlock()

	s.push(backfilling, inst)
}

func (s *Subscription) push(backfilling bool, inst *instructionWrapper) {
	if backfilling {
		zlog.Debug("sending backfill instruction to subscription",
			zap.Int("sub stream length", len(s.Stream)),
			zap.Int("cap stream length", cap(s.Stream)),
			zap.Reflect("instruction", inst),
		)
		s.Stream <- inst
		s.backfilledTrxSignatures[inst.TrxSignature] = true
		return
	}

	if s.backfillCompleted.Load() {
		zlog.Debug("sending live instruction to subscription",
			zap.Int("sub stream length", len(s.Stream)),
			zap.Int("cap stream length", cap(s.Stream)),
			zap.Reflect("instruction", inst),
		)
		s.Stream <- inst
		return
	}

	s.toSendLiveInstructions = append(s.toSendLiveInstructions, inst)
	return
}

func (s *Subscription) Backfill(ctx context.Context, rpcClient *rpc.Client) {
	t0 := time.Now()
	zlog.Info("back filling subscription",
		zap.Time("started_at", t0),
	)
	transaction.GetTransactionForAccount(ctx, rpcClient, s.account, func(trx *rpc.TransactionWithMeta) {
		zlog.Debug("got a transaction",
			zap.String("signature", trx.Transaction.Signatures[0].String()),
		)
		if !trx.Transaction.IsSigner(s.account) {
			zlog.Debug("transaction was not signed by subscribed account")
			return
		}
		zlog.Debug("getting instruction for transaction")
		getStreamableInstructions(trx, func(compiledInstruction *solana.CompiledInstruction, decodedInstruction *serum.Instruction) {
			s.pushSafe(true, &instructionWrapper{
				Decoded:      decodedInstruction,
				Compiled:     compiledInstruction,
				TrxError:     trx.Meta.Err,
				TrxSignature: trx.Transaction.Signatures[0].String(),
			})
		})
	})

	s.pushLock.Lock()
	defer s.pushLock.Unlock()
	zlog.Info("backfilling completed draining pending live instruction queue",
		zap.Int("queue_size", len(s.toSendLiveInstructions)),
		zap.Duration("backfill_duration", time.Since(t0)),
	)
	for _, inst := range s.toSendLiveInstructions {
		s.push(true, inst)
	}
	s.backfillCompleted.Store(true)
}

type Manager struct {
	sync.RWMutex

	subscriptions map[string][]*Subscription
}

func (m *Manager) ProcessErr(err error) {
	if traceEnabled {
		zlog.Debug("manager received stream err", zap.String("error", err.Error()))
	}
}

func NewManager() *Manager {
	return &Manager{
		RWMutex:       sync.RWMutex{},
		subscriptions: map[string][]*Subscription{},
	}
}

func (m *Manager) Process(trx *rpc.TransactionWithMeta) {
	m.RLock()

	subscriptions := []*Subscription{}
	for acc, subs := range m.subscriptions {
		if trx.Transaction.IsSigner(solana.MustPublicKeyFromBase58(acc)) {
			subscriptions = append(subscriptions, subs...)
		}
	}
	m.RUnlock()

	if len(subscriptions) == 0 {
		return
	}

	getStreamableInstructions(trx, func(compiledInstruction *solana.CompiledInstruction, decodedInstruction *serum.Instruction) {
		for _, sub := range subscriptions {
			sub.pushSafe(false, &instructionWrapper{
				Decoded:      decodedInstruction,
				Compiled:     compiledInstruction,
				TrxError:     trx.Meta.Err,
				TrxSignature: trx.Transaction.Signatures[0].String(),
			})
		}
	})
}

func getStreamableInstructions(trx *rpc.TransactionWithMeta, sender func(compiledInst *solana.CompiledInstruction, inst *serum.Instruction)) {
	for idx, compiledInstruction := range trx.Transaction.Message.Instructions {
		programID, err := trx.Transaction.ResolveProgramIDIndex(compiledInstruction.ProgramIDIndex)
		if err != nil {
			zlog.Info("invalid programID index... weird")
			continue
		}

		if programID.Equals(serum.DEX_PROGRAM_ID) {
			decodedInstruction, err := serum.DecodeInstruction(trx.Transaction.Message.AccountKeys, &compiledInstruction)
			if err != nil {
				zlog.Error("error decoding instruction",
					zap.Error(err),
					zap.Stringer("trx_signature", trx.Transaction.Signatures[0]),
					zap.Int("instruction_index", idx),
					zap.String("data", hex.EncodeToString(compiledInstruction.Data)),
				)
				sender(&compiledInstruction, nil)
				return
			}

			sender(&compiledInstruction, decodedInstruction)
		} else {
			if traceEnabled {
				zlog.Debug("skipping none serum DEX program ID",
					zap.Stringer("program_id", programID),
				)
			}
		}
	}
}

func (m *Manager) Subscribe(sub *Subscription) {
	m.Lock()
	defer m.Unlock()
	m.subscriptions[sub.account.String()] = append(m.subscriptions[sub.account.String()], sub)
	zlog.Info("subscribed",
		zap.Stringer("account", sub.account),
		zap.Int("new_length", len(m.subscriptions[sub.account.String()])),
	)
}

func (m *Manager) Unsubscribe(toRemove *Subscription) bool {
	m.Lock()
	defer m.Unlock()
	accountStr := toRemove.account.String()
	if subs, ok := m.subscriptions[accountStr]; ok {
		var newListeners []*Subscription
		for _, sub := range subs {
			if sub != toRemove {
				newListeners = append(newListeners, sub)
			}
		}
		m.subscriptions[accountStr] = newListeners
		zlog.Info("unsubscribed",
			zap.Stringer("account", toRemove.account),
			zap.Int("new_length", len(newListeners)),
		)
	}
	return true
}
