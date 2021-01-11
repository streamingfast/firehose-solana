package md

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	bin "github.com/dfuse-io/binary"
	"go.uber.org/zap"

	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/token"
)

type RegisteredToken struct {
	*token.Mint
	Meta    *TokenMeta
	Address solana.PublicKey
}

type TokenMeta struct {
	Logo    string
	Name    string
	Symbol  string
	Website string
}

type Token struct {
	*token.Mint
	Meta    *TokenMeta
	Address solana.PublicKey
}

type tokenJob struct {
	Name   string           `json:"tokenName"`
	Symbol string           `json:"tokenSymbol"`
	Mint   solana.PublicKey `json:"mintAddress"`
	Icon   string           `json:"icon"`
}

func (s *Server) readKnownTokens() error {
	jobs := make(chan *tokenJob, 1000)

	var wg sync.WaitGroup
	for w := 1; w <= 20; w++ {
		wg.Add(1)
		go s.addToken(w, &wg, jobs)
	}

	err := readFile(s.tokenListURL, func(line string) error {
		var t *tokenJob
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			return fmt.Errorf("unable decode token inforation: %w", err)
		}
		jobs <- t
		return nil
	})

	close(jobs)
	if err != nil {
		return err
	}

	wg.Wait()
	zlog.Info("known tokens loaded",
		zap.Int("token_count", len(s.tokenStore)),
	)
	return nil
}

func (s *Server) addToken(id int, wg *sync.WaitGroup, jobs <-chan *tokenJob) {
	defer wg.Done()

	for j := range jobs {
		zlog.Debug("retrieving known token",
			zap.String("name", j.Name),
			zap.String("symbol", j.Symbol),
		)
		res, err := s.rpcClient.GetAccountInfo(context.Background(), j.Mint)
		if err != nil {
			zlog.Warn("unable to retrieve token account",
				zap.Stringer("mint_address", j.Mint),
				zap.Error(err),
			)
			continue
		}

		var mint *token.Mint
		if err := bin.NewDecoder(res.Value.Data).Decode(&mint); err != nil {
			zlog.Warn("unable to retrieve token account",
				zap.Stringer("mint_address", j.Mint),
				zap.Error(err),
			)
			continue
		}
		s.tokenStoreLock.Lock()
		s.tokenStore[j.Mint.String()] = &RegisteredToken{
			Mint: mint,
			Meta: &TokenMeta{
				Name:   j.Name,
				Symbol: j.Symbol,
			},
			Address: solana.PublicKey{},
		}
		s.tokenStoreLock.Unlock()
	}
}
