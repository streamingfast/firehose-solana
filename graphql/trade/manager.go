package trade

import (
	"github.com/dfuse-io/solana-go"
	"sync"

	"github.com/dfuse-io/solana-go/rpc"

	"go.uber.org/zap"
)

type Subscription struct {
	Stream  chan *Trade
	account string
}

func newSubscription(account string) *Subscription {
	return &Subscription{
		account: account,
		Stream:  make(chan *Trade, 200),
	}
}

type Manager struct {
	sync.RWMutex

	subscriptions map[string][]*Subscription
}

func NewManager() *Manager {
	return &Manager{
		RWMutex:       sync.RWMutex{},
		subscriptions: map[string][]*Subscription{},
	}
}

func (m *Manager) Process(trx *rpc.TransactionWithMeta) {
	accounts := trx.Transaction.Message.AccountKeys


	matchedAccounts := map[string]Subscription{}
	for _, a := range accounts {
		accountAddress := a.String()
		if subs, found := m.subscriptions[a.String()]; found {
			matchedAccounts[accountAddress] = subs
		}
	}
	if !matched {
		return
	}

	for _, i := range trx.Transaction.Message.Instructions {
		i.
	}
}
func (m *Manager) ProcessErr(err error) {

}

func (m *Manager) Subscribe(account string) *Subscription {
	m.Lock()
	defer m.Unlock()

	sub := newSubscription(account)

	m.subscriptions[account] = append(m.subscriptions[account], sub)
	zlog.Info("subscribed",
		zap.String("account", account),
		zap.Int("new_length", len(m.subscriptions[account])),
	)
	return sub
}

func (m *Manager) Unsubscribe(toRemove *Subscription) bool {
	m.Lock()
	defer m.Unlock()
	if subs, ok := m.subscriptions[toRemove.account]; ok {
		var newListeners []*Subscription
		for _, sub := range subs {
			if sub != toRemove {
				newListeners = append(newListeners, sub)
			}
		}
		m.subscriptions[toRemove.account] = newListeners
		zlog.Info("unsubscribed",
			zap.String("account", toRemove.account),
			zap.Int("new_length", len(newListeners)),
		)
	}
	return true
}
