package registry

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/dfuse-io/solana-go"
)

type Market struct {
	Name         string           `json:"name"`
	Address      solana.PublicKey `json:"address"`
	Deprecated   bool             `json:"deprecated"`
	ProgramID    solana.PublicKey `json:"program_id"`
	BaseToken    solana.PublicKey `json:"base_token"`
	QuoteToken   solana.PublicKey `json:"quote_token"`
	BaseLotSize  uint64           `json:"base_lot_size"`
	QuoteLotSize uint64           `json:"quote_lot_size"`
	RequestQueue solana.PublicKey `json:"request_queue"`
	EventQueue   solana.PublicKey `json:"event_queue"`
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

func ReadKnownMarkets(ctx context.Context, marketListURL string) (map[string]*Market, error) {
	out := map[string]*Market{}

	err := readFile(ctx, marketListURL, func(line string) error {
		var m *Market
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return fmt.Errorf("unable decode market information: %w", err)
		}
		out[m.Address.String()] = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
