package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dfuse-io/solana-go/rpc"

	"github.com/dfuse-io/solana-go/programs/token"

	"go.uber.org/zap"

	"github.com/dfuse-io/solana-go"
)

type Token struct {
	Address               solana.PublicKey `json:"address"`
	MintAuthorityOption   uint32           `json:"mint_authority_option"`
	MintAuthority         solana.PublicKey `json:"mint_authority"`
	Supply                uint64           `json:"supply"`
	Decimals              uint8            `json:"decimals"`
	IsInitialized         bool             `json:"is_initialized"`
	FreezeAuthorityOption uint32           `json:"freeze_authority_option"`
	FreezeAuthority       solana.PublicKey `json:"freeze_authority"`
	Verified              bool             `json:"verified"`
	Meta                  *TokenMeta       `json:"meta"`
}

func (t *Token) Display(amount uint64) string {
	var symbol string
	if t.Meta != nil {
		symbol = t.Meta.Symbol
	} else {
		key := t.Address.String()
		symbol = fmt.Sprintf("%s..%s", key[0:4], key[len(key)-4:])
	}

	v := F().Quo(F().SetUint64(amount), F().SetInt(decimalMultiplier(uint(t.Decimals))))
	return fmt.Sprintf("%f %s", v, symbol)
}

type TokenMeta struct {
	Logo    string
	Name    string
	Symbol  string
	Website string
}

type tokenJob struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Icon   string `json:"icon"`
}

func (s *Server) GetToken(address *solana.PublicKey) *Token {
	s.tokenStoreLock.RLock()
	defer s.tokenStoreLock.RUnlock()

	return s.tokenStore[address.String()]
}

func (s *Server) GetTokens() (out []*Token) {
	s.tokenStoreLock.RLock()
	defer s.tokenStoreLock.RUnlock()

	zlog.Info("get tokens",
		zap.Int("store_size", len(s.tokenStore)),
	)

	out = []*Token{}
	for _, t := range s.tokenStore {
		out = append(out, t)
	}
	return
}

func ReadKnownTokens(ctx context.Context, tokenListURL string) (map[string]*Token, error) {
	out := map[string]*Token{}

	err := readFile(ctx, tokenListURL, func(line string) error {
		var t *Token
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			return fmt.Errorf("unable decode market information: %w", err)
		}
		out[t.Address.String()] = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func SyncKnownTokens(rpcClient *rpc.Client, tokens []*Token) (out []*Token, err error) {
	jobs := make(chan *Token, 1000)
	results := make(chan *Token, 1000)
	var wg sync.WaitGroup
	for w := 1; w <= 20; w++ {
		wg.Add(1)
		go processToken(&wg, rpcClient, jobs, results)
	}

	for _, t := range tokens {
		jobs <- t
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for {
		select {
		case t, ok := <-results:
			if !ok {
				return out, nil
			}
			out = append(out, t)
		}
	}
}

func processToken(wg *sync.WaitGroup, rpc *rpc.Client, jobs <-chan *Token, results chan *Token) {
	defer wg.Done()

	for tk := range jobs {
		zlog.Debug("retrieving known token",
			zap.Stringer("mint_address", tk.Address),
		)
		res, err := rpc.GetAccountInfo(context.Background(), tk.Address)
		if err != nil {
			zlog.Warn("unable to retrieve token account",
				zap.Stringer("mint_address", tk.Address),
			)
			continue
		}

		mint := &token.Mint{}
		if err := mint.Decode(res.Value.Data); err != nil {
			zlog.Warn("unable to retrieve token account",
				zap.Stringer("mint_address", tk.Address),
				zap.Error(err),
			)
			continue
		}

		tk.MintAuthorityOption = mint.MintAuthorityOption
		tk.MintAuthority = mint.MintAuthority
		tk.Supply = uint64(mint.Supply)
		tk.Decimals = mint.Decimals
		tk.IsInitialized = mint.IsInitialized
		tk.FreezeAuthorityOption = mint.FreezeAuthorityOption
		tk.FreezeAuthority = mint.FreezeAuthority
		results <- tk
	}
}
