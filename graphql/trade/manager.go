package trade

import (
	"context"
	"encoding/hex"
	"sync"

	"github.com/dfuse-io/dfuse-solana/transaction"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/serum"
	"go.uber.org/zap"
)

type instructionWrapper struct {
	Inst  *serum.Instruction
	TrxID string
}

type Subscription struct {
	Stream  chan *instructionWrapper
	account solana.PublicKey
	Err     error
}

func (s Subscription) Push(inst *instructionWrapper) {
	zlog.Debug("sending instruction to subscription",
		zap.Int("sub stream length", len(s.Stream)),
		zap.Int("cap stream length", cap(s.Stream)),
		zap.Reflect("instruction", inst),
	)

	s.Stream <- inst
}

func (s *Subscription) Backfill(ctx context.Context, rpcClient *rpc.Client) {
	zlog.Info("back filling subscription")
	transaction.GetTransactionForAccount(ctx, rpcClient, s.account, func(trx *rpc.TransactionWithMeta) {
		zlog.Debug("got a transaction", zap.String("signature", trx.Transaction.Signatures[0].String()))
		if !trx.Transaction.IsSigner(s.account) {
			zlog.Debug("transaction was not signed by subscribed account")
			return
		}
		zlog.Debug("getting instruction for transaction")
		getStreamableInstructions(trx, func(inst *serum.Instruction, err error) {
			if err != nil { //send error to all sub ...
				s.Err = err
				close(s.Stream)
				return
			}
			zlog.Debug("got instruction")
			s.Push(&instructionWrapper{
				Inst:  inst,
				TrxID: trx.Transaction.Signatures[0].String(),
			})
		})
	})
	zlog.Info("back fill terminated")
}

func NewSubscription(account solana.PublicKey) *Subscription {
	return &Subscription{
		account: account,
		Stream:  make(chan *instructionWrapper, 200),
	}
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

	getStreamableInstructions(trx, func(inst *serum.Instruction, err error) {
		for _, sub := range subscriptions {
			if err != nil { //send error to all sub ...
				sub.Err = err
				close(sub.Stream)
				continue
			}
			sub.Push(&instructionWrapper{
				Inst:  inst,
				TrxID: trx.Transaction.Signatures[0].String(),
			})
		}
	})
}

func getStreamableInstructions(trx *rpc.TransactionWithMeta, sender func(inst *serum.Instruction, err error)) {
	for idx, ins := range trx.Transaction.Message.Instructions {
		programID, err := trx.Transaction.ResolveProgramIDIndex(ins.ProgramIDIndex)
		if err != nil {
			zlog.Info("invalid programID index... werid")
			continue
		}

		if programID.Equals(serum.DEX_PROGRAM_ID) {
			instruction, err := serum.DecodeInstruction(trx.Transaction.Message.AccountKeys, &ins)
			if err != nil {
				zlog.Error("error decoding instruction",
					zap.Error(err),
					zap.Stringer("trx_signature", trx.Transaction.Signatures[0]),
					zap.Int("instruction_index", idx),
					zap.String("data", hex.EncodeToString(ins.Data)),
				)
				sender(nil, err)
				return
			}

			sender(instruction, nil)
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
