package graphql

import (
	"go.uber.org/zap"
	"sync"
)

type AccountSubscription struct {
	Stream  chan *Trade
	account string
}

func newSubscription(account string) *AccountSubscription {
	return &AccountSubscription{
		account: account,
		Stream:  make(chan *Trade, 200),
	}
}

type Manager struct {
	sync.RWMutex

	subscriptions map[string][]*AccountSubscription
}

func NewManager() *Manager {
	return &Manager{
		RWMutex:       sync.RWMutex{},
		subscriptions: nil,
	}
}

func (m *Manager) ProcessBlock(blk interface{}) {
	founc : =map[]

	Transaction()
	Tran

	sub.Push(NewTrade)

}

func (m *Manager) Subscribe(account string) *AccountSubscription {
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

func (m *Manager) Unsubscribe(toRemove *AccountSubscription) bool {
	m.Lock()
	defer m.Unlock()
	if subs, ok := m.subscriptions[toRemove.account]; ok {
		var newListeners []*AccountSubscription
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
