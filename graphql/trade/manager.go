package trade

import (
	"context"
	"encoding/hex"
	"sync"

	"github.com/dfuse-io/dfuse-solana/transaction"

	"github.com/dfuse-io/solana-go"

	"github.com/dfuse-io/solana-go/serum"

	"github.com/dfuse-io/solana-go/rpc"
	"go.uber.org/zap"
)

type Subscription struct {
	Stream  chan *serum.Instruction
	account solana.PublicKey
}

func (s Subscription) Push(inst *serum.Instruction) {
	zlog.Debug("sending instruction to subscription",
		zap.Reflect("instruction", inst),
	)
	// todo should we check channel capacity
	s.Stream <- inst
}

func (s *Subscription) Backfill(ctx context.Context, rpcClient *rpc.Client) {
	transaction.GetTransactionForAccount(ctx, rpcClient, s.account, func(trx *rpc.TransactionWithMeta) {
		if !trx.Transaction.IsSigner(s.account) {
			return
		}
		getStreamableInstructions(trx, func(inst *serum.Instruction) {
			s.Push(inst)
		})
	})
}

func NewSubscription(account solana.PublicKey) *Subscription {
	return &Subscription{
		account: account,
		Stream:  make(chan *serum.Instruction, 200),
	}
}

type Manager struct {
	sync.RWMutex

	subscriptions map[string][]*Subscription
}

func (m *Manager) ProcessErr(err error) {
	zlog.Debug("managaer received stream err", zap.String("error", err.Error()))
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

	getStreamableInstructions(trx, func(inst *serum.Instruction) {
		for _, sub := range subscriptions {
			sub.Push(inst)
		}
	})
}

func getStreamableInstructions(trx *rpc.TransactionWithMeta, sender func(inst *serum.Instruction)) {
	for idx, ins := range trx.Transaction.Message.Instructions {
		programID, err := trx.Transaction.ResolveProgramIDIndex(ins.ProgramIDIndex)
		if err != nil {
			zlog.Info("invalid programID index... werid")
			continue
		}

		if programID.Equals(serum.DEX_PROGRAM_ID) {
			instruction, err := serum.DecodeInstruction(&ins)
			if err != nil {
				zlog.Error("error decoding instruction",
					zap.Error(err),
					zap.Stringer("trx_signature", trx.Transaction.Signatures[0]),
					zap.Int("instruction_index", idx),
					zap.String("data", hex.EncodeToString(ins.Data)),
				)
				continue
			}

			sender(instruction)
		} else {
			zlog.Debug("skipping none serum DEX program ID",
				zap.Stringer("program_id", programID),
			)
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
