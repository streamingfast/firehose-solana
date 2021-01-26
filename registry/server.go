package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/dfuse-io/solana-go/rpc"
	"go.uber.org/zap"
)

type Server struct {
	tokenListURL   string
	tokenStore     map[string]*Token
	tokenStoreLock sync.RWMutex

	marketListURL   string
	marketStore     map[string]*Market
	marketStoreLock sync.RWMutex

	wsURL     string
	rpcClient *rpc.Client
}

func NewServer(rpcClient *rpc.Client, tokenListURL string, marketListURL string, wsURL string) *Server {
	return &Server{
		rpcClient:     rpcClient,
		tokenListURL:  tokenListURL,
		marketListURL: marketListURL,
		wsURL:         wsURL,
		tokenStore:    map[string]*Token{},
		marketStore:   map[string]*Market{},
	}
}

func (s *Server) Launch(ctx context.Context) (err error) {
	zlog.Info("loading known tokens")
	tokens, err := ReadKnownTokens(ctx, s.tokenListURL)
	if err != nil {
		return fmt.Errorf("unable to load known tokens: %w", err)
	}
	s.tokenStoreLock.Lock()
	s.tokenStore = tokens
	s.tokenStoreLock.Unlock()
	zlog.Info("known tokens loaded",
		zap.Int("count", len(s.tokenStore)),
	)

	zlog.Info("loading known markets")
	markets, err := ReadKnownMarkets(ctx, s.marketListURL)
	if err != nil {
		return fmt.Errorf("unable to load known markets: %w", err)
	}
	s.marketStoreLock.Lock()
	s.marketStore = markets
	s.marketStoreLock.Unlock()
	zlog.Info("known markets loaded",
		zap.Int("count", len(s.marketStore)),
	)

	return nil
}
