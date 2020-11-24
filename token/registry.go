package token

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dfuse-io/solana-go/rpc/ws"

	"go.uber.org/zap"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go/token"

	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
)

type RegistredToken struct {
	*token.Mint
	Symbol  string
	Address solana.PublicKey
}

type Registry struct {
	names     map[string]string
	rpcClient *rpc.Client
	store     map[string]*RegistredToken
	storeLock sync.RWMutex
	wsURL     string
}

func NewRegistry(rpcClient *rpc.Client, wsURL string) *Registry {
	return &Registry{
		rpcClient: rpcClient,
		wsURL:     wsURL,
		names:     map[string]string{},
		store:     map[string]*RegistredToken{},
	}
}

func (r *Registry) GetToken(address *solana.PublicKey) *RegistredToken {
	r.storeLock.RLock()
	defer r.storeLock.RUnlock()

	return r.store[address.String()]
}

func (r *Registry) GetTokens() (out []*RegistredToken) {
	r.storeLock.RLock()
	defer r.storeLock.RUnlock()

	out = []*RegistredToken{}
	for _, t := range r.store {
		out = append(out, t)
	}
	zlog.Info("about to return tokens", zap.Int("count", len(out)))
	return
}

func (r *Registry) loadNames() error {
	var nameList []struct {
		Address string
		Name    string
	}

	if err := json.Unmarshal([]byte(jsonData), &nameList); err != nil {
		return fmt.Errorf("load names: %w", err)
	}

	for _, n := range nameList {
		r.names[n.Address] = n.Name
	}
	return nil
}

func (r *Registry) Load() error {
	address := "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	pubKey := solana.MustPublicKeyFromBase58(address)

	wsClient, err := ws.Dial(context.Background(), r.wsURL)
	if err != nil {
		return fmt.Errorf("loading: ws dial: %e", err)
	}
	go r.watch(pubKey, wsClient)

	if err := r.loadNames(); err != nil {
		return fmt.Errorf("loading: name: %w", err)
	}

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
			r.store[aPub] = &RegistredToken{
				Address: a.Pubkey,
				Mint:    mint,
				Symbol:  r.names[aPub],
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
				r.store[addr] = &RegistredToken{
					Address: address,
					Mint:    mint,
					Symbol:  r.names[addr],
				}
				r.storeLock.Unlock()
			} else {
				zlog.Debug("skipping program update, not a mint account")
			}
		}
	}
}
