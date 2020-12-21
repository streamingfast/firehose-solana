package token

import (
	"context"
	"fmt"
	"sync"
	"time"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/token"
	"github.com/dfuse-io/solana-go/programs/tokenregistry"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/rpc/ws"
	"go.uber.org/zap"
)

type TokenMeta struct {
	Logo    string
	Name    string
	Symbol  string
	Website string
}

type RegisteredToken struct {
	*token.Mint
	Meta    *TokenMeta
	Address solana.PublicKey
}

type Registry struct {
	rpcClient *rpc.Client
	store     map[string]*RegisteredToken
	storeLock sync.RWMutex
	wsURL     string
}

func NewRegistry(rpcClient *rpc.Client, wsURL string) *Registry {
	return &Registry{
		rpcClient: rpcClient,
		wsURL:     wsURL,
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

func (r *Registry) Load() (err error) {
	address := token.TOKEN_PROGRAM_ID.String()
	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return fmt.Errorf("unable to create public key from token program id address %q: %w", address, err)
	}

	var metas map[string]*TokenMeta
	if metas, err = r.loadMetas(); err != nil {
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
				Meta:    metas[aPub],
			}
		}
		r.storeLock.Unlock()
	}
	return nil
}

func (r *Registry) loadMetas() (out map[string]*TokenMeta, err error) {
	zlog.Info("loading token meta from chain registry")
	out = map[string]*TokenMeta{}

	progID := tokenregistry.ProgramID()
	wsClient, err := ws.Dial(context.Background(), r.wsURL)
	if err != nil {
		return nil, fmt.Errorf("loading meta: ws dial: %e", err)
	}

	go r.watchMeta(wsClient)

	accounts, err := r.rpcClient.GetProgramAccounts(context.Background(), progID, nil)
	if err != nil {
		return nil, fmt.Errorf("loading metas: get program accounts: %s : %w", progID, err)
	}
	if accounts == nil {
		return nil, fmt.Errorf("loading metas: get program accounts: not found for: %s", progID)
	}

	zlog.Info("found token meta", zap.Int("count", len(accounts)))
	for _, a := range accounts {
		var m *tokenregistry.TokenMeta
		if err := bin.NewDecoder(a.Account.Data).Decode(&m); err != nil {
			return nil, fmt.Errorf("loading meta: get program accounts: decoding to Token meta: %s", a.Account.Data)
		}
		zlog.Info("storing meta", zap.Stringer("token_meta_address", a.Pubkey), zap.Stringer("mint_address", m.MintAddress))

		out[a.Pubkey.String()] = &TokenMeta{
			Symbol:  m.Symbol.String(),
			Name:    m.Name.String(),
			Logo:    m.Logo.String(),
			Website: m.Website.String(),
		}
	}
	return
}

func (r *Registry) watchMeta(client *ws.Client) {
	progID := tokenregistry.ProgramID()
	zlog.Info("watching metas ", zap.Stringer("address", progID))
	sleep := 0 * time.Second

retry:
	for {
		time.Sleep(sleep)
		sleep = 2 * time.Second
		zlog.Info("watching meta: getting program subscription", zap.Stringer("program_address", progID))
		sub, err := client.ProgramSubscribe(progID, rpc.CommitmentSingle)
		if err != nil {
			zlog.Error("failed to subscribe", zap.Stringer("address", progID))
			continue
		}

		for {
			res, err := sub.Recv()
			if err != nil {
				zlog.Error("failed to receive from subscribe", zap.Error(err))
				continue retry
			}
			zlog.Info("watching meta: received message")
			programResult := res.(*ws.ProgramResult)
			var m *tokenregistry.TokenMeta
			if err := bin.NewDecoder(programResult.Value.Account.Data).Decode(&m); err != nil {
				zlog.Error("decoding", zap.Error(err))
				continue retry
			}
			mintAddress := m.MintAddress.String()
			tokenMeta := &TokenMeta{
				Symbol:  m.Symbol.String(),
				Name:    m.Name.String(),
				Logo:    m.Logo.String(),
				Website: m.Logo.String(),
			}

			zlog.Info("Updating token meta", zap.String("token_address", mintAddress))
			r.storeLock.Lock()

			if storedToken, found := r.store[mintAddress]; found {
				storedToken.Meta = tokenMeta
			} else {
				zlog.Warn("found meta for a unknown token", zap.String("mint_address", mintAddress), zap.Stringer("meta_address", programResult.Value.PubKey))
			}
			r.storeLock.Unlock()
		}
	}
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
				if _, found := r.store[addr]; !found {
					r.store[addr] = &RegisteredToken{
						Address: address,
						Mint:    mint,
					}
				}
				r.storeLock.Unlock()
			} else {
				zlog.Debug("skipping program update, not a mint account")
			}
		}
	}
}
