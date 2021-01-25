package registry

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/dfuse-io/solana-go"
)

type Market struct {
	Name       string           `json:"name"`
	Address    solana.PublicKey `json:"address"`
	Deprecated bool             `json:"deprecated"`
	ProgramID  solana.PublicKey `json:"program_id"`
	BaseToken  solana.PublicKey `json:"base_token"`
	QuoteToken solana.PublicKey `json:"quote_token"`
}

func (s *Server) GetMarket(address *solana.PublicKey) *Market {
	s.marketStoreLock.RLock()
	defer s.marketStoreLock.RUnlock()

	if m, found := s.marketStore[address.String()]; found {
		return m
	}
	return nil
}

func (s *Server) GetMarkets() (out []*Market) {
	s.marketStoreLock.RLock()
	defer s.marketStoreLock.RUnlock()

	zlog.Info("get markets",
		zap.Int("store_size", len(s.tokenStore)),
	)

	out = []*Market{}
	for _, t := range s.marketStore {
		out = append(out, t)
	}
	return
}

func (s *Server) readKnownMarkets() error {
	err := readFile(s.marketListURL, func(line string) error {
		var m *Market
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return fmt.Errorf("unable decode market information: %w", err)
		}
		s.marketStoreLock.Lock()
		s.marketStore[m.Address.String()] = m
		s.marketStoreLock.Unlock()
		return nil
	})
	if err != nil {
		return err
	}

	zlog.Info("known markets loaded",
		zap.Int("market_count", len(s.marketStore)),
	)
	return nil
}
