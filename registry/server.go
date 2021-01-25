package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dfuse-io/solana-go/programs/tokenregistry"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/token"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/rpc/ws"
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

func (s *Server) Launch(loadFromChain bool) (err error) {
	zlog.Info("loading known tokens")
	if err := s.readKnownTokens(); err != nil {
		return fmt.Errorf("unable to receive known tokens: %w", err)
	}

	zlog.Info("loading known markets")
	if err := s.readKnownMarkets(); err != nil {
		return fmt.Errorf("unable to load known markets: %w", err)
	}

	if loadFromChain {
		return s.loadFromChain()
	}
	return nil
}

func (s *Server) loadFromChain() error {
	wsClient, err := ws.Dial(context.Background(), s.wsURL)
	if err != nil {
		return fmt.Errorf("loading meta: ws dial: %e", err)
	}

	zlog.Info("loading tokens from chain",
		zap.Stringer("token_program", token.TOKEN_PROGRAM_ID),
	)

	if err := s.loadChainTokens(wsClient); err != nil {
		return fmt.Errorf("loading chain tokens: %w", err)
	}

	if err := s.loadChainTokenRegistry(wsClient); err != nil {
		return fmt.Errorf("loading chain tokens: %w", err)
	}
	return nil
}

func (s *Server) saveTokenMeta(mint *solana.PublicKey, tokenMeta *TokenMeta) {
	s.tokenStoreLock.Lock()
	if storedToken, found := s.tokenStore[mint.String()]; found {
		storedToken.Meta = tokenMeta
	}
	s.tokenStoreLock.Unlock()
}

func (s *Server) loadChainTokenRegistry(client *ws.Client) error {
	zlog.Info("loading tokens from chain",
		zap.Stringer("token_registry_address", tokenregistry.ProgramID()),
	)

	go rpcWatchAddress(client, tokenregistry.ProgramID(), func(result *ws.ProgramResult) error {
		zlog.Info("watching meta: received message")
		var m *tokenregistry.TokenMeta
		if err := bin.NewDecoder(result.Value.Account.Data).Decode(&m); err != nil {
			return fmt.Errorf("unable to decode token reigistry account: %w", err)
		}

		mintAddress := m.MintAddress.String()
		tokenMeta := &TokenMeta{
			Symbol:  m.Symbol.String(),
			Name:    m.Name.String(),
			Logo:    m.Logo.String(),
			Website: m.Logo.String(),
		}

		zlog.Info("Updating token meta", zap.String("token_address", mintAddress))

		s.saveTokenMeta(m.MintAddress, tokenMeta)
		return nil
	})

	accounts, err := s.rpcClient.GetProgramAccounts(context.Background(), tokenregistry.ProgramID(), nil)
	if err != nil {
		return fmt.Errorf("loading metas: get program accounts: %s : %w", tokenregistry.ProgramID().String(), err)
	}
	if accounts == nil {
		return fmt.Errorf("loading metas: get program accounts: not found for: %s", tokenregistry.ProgramID().String())
	}

	zlog.Info("found token meta", zap.Int("count", len(accounts)))
	for _, a := range accounts {
		var m *tokenregistry.TokenMeta
		if err := bin.NewDecoder(a.Account.Data).Decode(&m); err != nil {
			return fmt.Errorf("loading meta: get program accounts: decoding to Token meta: %s", a.Account.Data)
		}
		zlog.Info("storing meta", zap.Stringer("token_meta_address", a.Pubkey), zap.Stringer("mint_address", m.MintAddress))

		s.saveTokenMeta(m.MintAddress, &TokenMeta{
			Symbol:  m.Symbol.String(),
			Name:    m.Name.String(),
			Logo:    m.Logo.String(),
			Website: m.Website.String(),
		})
	}
	return nil
}

func (s *Server) loadChainTokens(client *ws.Client) error {
	zlog.Info("loading tokens from chain",
		zap.Stringer("token_address", token.TOKEN_PROGRAM_ID),
	)

	go rpcWatchAddress(client, token.TOKEN_PROGRAM_ID, func(result *ws.ProgramResult) error {
		if len(result.Value.Account.Data) != 82 {
			zlog.Debug("skipping program update, not a mint account")
			return nil
		}

		var mint *token.Mint
		if err := bin.NewDecoder(result.Value.Account.Data).Decode(&mint); err != nil {
			return fmt.Errorf("unable to decode mint: %w", err)
		}

		addr := result.Value.PubKey.String()
		zlog.Info("Updating token",
			zap.String("token_address", addr),
			zap.Uint64("supply", uint64(mint.Supply)),
		)

		s.tokenStoreLock.Lock()
		if _, found := s.tokenStore[addr]; !found {
			s.tokenStore[addr] = &Token{
				Address: result.Value.PubKey,
				Mint:    mint,
			}
		}
		s.tokenStoreLock.Unlock()
		return nil
	})

	accounts, err := s.rpcClient.GetProgramAccounts(
		context.Background(),
		token.TOKEN_PROGRAM_ID,
		&rpc.GetProgramAccountsOpts{
			Filters: []rpc.RPCFilter{
				{
					DataSize: 82,
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("loading: get program accounts: %w", err)
	}
	if accounts == nil {
		return fmt.Errorf("loading: get program accounts: not found for: %s", token.TOKEN_PROGRAM_ID.String())
	}

	for _, a := range accounts {
		var mint *token.Mint
		if err := bin.NewDecoder(a.Account.Data).Decode(&mint); err != nil {
			return fmt.Errorf("loading: get program accounts: decoding: %s", a.Account.Data)
		}
		aPub := a.Pubkey.String()

		s.tokenStoreLock.Lock()
		if _, found := s.tokenStore[aPub]; !found { // can be found if the watch process added it
			s.tokenStore[aPub] = &Token{
				Address: a.Pubkey,
				Mint:    mint,
			}
		}
		s.tokenStoreLock.Unlock()
	}
	return nil
}

func rpcWatchAddress(client *ws.Client, address solana.PublicKey, f func(*ws.ProgramResult) error) {
	zlog.Info("watching address ",
		zap.Stringer("address", address),
	)
	sleep := 0 * time.Second

retry:
	for {
		time.Sleep(sleep)
		sleep = 2 * time.Second

		sub, err := client.ProgramSubscribe(address, rpc.CommitmentSingle)
		if err != nil {
			zlog.Error("failed to subscribe",
				zap.Stringer("address", address),
			)
			continue
		}

		for {
			res, err := sub.Recv()
			if err != nil {
				zlog.Error("failed to receive from subscribe", zap.Error(err))
				continue retry
			}

			programResult := res.(*ws.ProgramResult)
			err = f(programResult)
			if err != nil {
				zlog.Error("error processing results",
					zap.Stringer("address", address),
					zap.Error(err),
				)

			}
		}
	}

}
