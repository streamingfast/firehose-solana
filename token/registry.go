package token

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dfuse-io/solana-go/programs/tokenregistry"

	"github.com/dfuse-io/solana-go/programs/token"

	"github.com/dfuse-io/solana-go/rpc/ws"

	"go.uber.org/zap"

	bin "github.com/dfuse-io/binary"

	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
)

type TokenMeta struct {
	Logo   string
	Name   string
	Symbol string
}
type RegisteredToken struct {
	*token.Mint
	Meta    *TokenMeta
	Address solana.PublicKey
}

type Registry struct {
	metas     map[string]*TokenMeta
	rpcClient *rpc.Client
	store     map[string]*RegisteredToken
	storeLock sync.RWMutex
	wsURL     string
}

func NewRegistry(rpcClient *rpc.Client, wsURL string) *Registry {
	return &Registry{
		rpcClient: rpcClient,
		wsURL:     wsURL,
		metas:     map[string]*TokenMeta{},
		store:     map[string]*RegisteredToken{},
	}
}

func (r *Registry) GetToken(address *solana.PublicKey) *RegisteredToken {
	r.storeLock.RLock()
	defer r.storeLock.RUnlock()

	return r.store[address.String()]
}

func (r *Registry) GetTokens() (out []*RegisteredToken) {
	r.storeLock.RLock()
	defer r.storeLock.RUnlock()

	zlog.Info("get tokens", zap.Int("store_size", len(r.store)))

	out = []*RegisteredToken{}
	for _, t := range r.store {
		out = append(out, t)
	}
	zlog.Info("about to return tokens", zap.Int("count", len(out)))
	return
}

func (r *Registry) Load() error {
	address := "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	pubKey := solana.MustPublicKeyFromBase58(address)

	if err := r.loadNames(); err != nil {
		return fmt.Errorf("loading: name: %w", err)
	}

	wsClient, err := ws.Dial(context.Background(), r.wsURL)
	if err != nil {
		return fmt.Errorf("loading: ws dial: %e", err)
	}

	go r.watch(pubKey, wsClient)

	accounts, err := r.rpcClient.GetProgramAccounts(
		context.Background(),
		pubKey,
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
		return fmt.Errorf("loading: get program accounts: not found for: %s", address)
	}

	for _, a := range accounts {
		var mint *token.Mint
		if err := bin.NewDecoder(a.Account.Data).Decode(&mint); err != nil {
			return fmt.Errorf("loading: get program accounts: decoding: %s", a.Account.Data)
		}
		aPub := a.Pubkey.String()

		r.storeLock.Lock()
		if _, found := r.store[aPub]; !found { // can be found if the watch process added it
			r.store[aPub] = &RegisteredToken{
				Address: a.Pubkey,
				Mint:    mint,
				Meta:    r.metas[aPub],
			}
		}
		r.storeLock.Unlock()
	}
	return nil
}

func (r *Registry) watch(address solana.PublicKey, client *ws.Client) {
	zlog.Info("watching token ", zap.Stringer("address", address))
	sleep := 0 * time.Second

retry:
	for {
		time.Sleep(sleep)
		sleep = 2 * time.Second
		zlog.Info("getting program subscription", zap.Stringer("program_address", address))
		sub, err := client.ProgramSubscribe(address, rpc.CommitmentSingle)
		if err != nil {
			zlog.Error("failed to subscribe", zap.Stringer("address", address))
			continue
		}

		for {
			res, err := sub.Recv()
			if err != nil {
				zlog.Error("failed to receive from subscribe", zap.Error(err))
				continue retry
			}

			programResult := res.(*ws.ProgramResult)
			if len(programResult.Value.Account.Data) == 82 {
				var mint *token.Mint
				if err := bin.NewDecoder(programResult.Value.Account.Data).Decode(&mint); err != nil {
					zlog.Error("decoding", zap.Error(err))
					continue retry
				}
				addr := programResult.Value.PubKey.String()

				zlog.Info("Updating token", zap.String("token_address", addr), zap.Uint64("supply", uint64(mint.Supply)))
				r.storeLock.Lock()
				r.store[addr] = &RegisteredToken{
					Address: address,
					Mint:    mint,
					Meta:    r.metas[addr],
				}
				r.storeLock.Unlock()
			} else {
				zlog.Debug("skipping program update, not a mint account")
			}
		}
	}
}

func (r *Registry) loadNames() error {
	var nameList []struct {
		Address string
		Name    string
	}

	if err := json.Unmarshal([]byte(jsonData), &nameList); err != nil {
		return fmt.Errorf("load metas: %w", err)
	}

	for _, n := range nameList {
		r.metas[n.Address] = &TokenMeta{
			Symbol: n.Name,
		}
	}
	return nil
}

func (r *Registry) loadMetas() error {

	wsClient, err := ws.Dial(context.Background(), r.wsURL)
	if err != nil {
		return fmt.Errorf("loading meta: ws dial: %e", err)
	}

	go r.watch(token.TOKEN_PROGRAM_ID, wsClient)

	accounts, err := r.rpcClient.GetProgramAccounts(context.Background(), tokenregistry.PROGRAM_ID, nil)
	if err != nil {
		return fmt.Errorf("loading metas: get program accounts: %s : %w", tokenregistry.PROGRAM_ID, err)
	}
	if accounts == nil {
		return fmt.Errorf("loading metas: get program accounts: not found for: %s", tokenregistry.PROGRAM_ID)
	}

	for _, a := range accounts {
		var m *tokenregistry.TokenMeta
		if err := bin.NewDecoder(a.Account.Data).Decode(&m); err != nil {
			return fmt.Errorf("loading meta: get program accounts: decoding to Token meta: %s", a.Account.Data)
		}
		r.metas[a.Pubkey.String()] = &TokenMeta{
			Symbol: m.Symbol.String(),
			Name:   m.Name.String(),
			Logo:   m.Logo.String(),
		}
	}

	return nil
}

func (r *Registry) watchMeta(client *ws.Client) {
	address := tokenregistry.PROGRAM_ID
	zlog.Info("watching metas ", zap.Stringer("address", address))
	sleep := 0 * time.Second

retry:
	for {
		time.Sleep(sleep)
		sleep = 2 * time.Second
		zlog.Info("getting program subscription", zap.Stringer("program_address", address))
		sub, err := client.ProgramSubscribe(address, rpc.CommitmentSingle)
		if err != nil {
			zlog.Error("failed to subscribe", zap.Stringer("address", address))
			continue
		}

		for {
			res, err := sub.Recv()
			if err != nil {
				zlog.Error("failed to receive from subscribe", zap.Error(err))
				continue retry
			}

			programResult := res.(*ws.ProgramResult)
			var m *tokenregistry.TokenMeta
			if err := bin.NewDecoder(programResult.Value.Account.Data).Decode(&m); err != nil {
				zlog.Error("decoding", zap.Error(err))
				continue retry
			}
			metaDataAddr := programResult.Value.PubKey.String()
			tokenMeta := &TokenMeta{
				Symbol: m.Symbol.String(),
				Name:   m.Name.String(),
				Logo:   m.Logo.String(),
			}

			zlog.Info("Updating token meta", zap.String("token_address", metaDataAddr))
			r.storeLock.Lock()
			r.store[metaDataAddr] = &RegisteredToken{
				Meta: tokenMeta,
			}
			r.metas[metaDataAddr] = tokenMeta

			r.storeLock.Unlock()
		}
	}
}
