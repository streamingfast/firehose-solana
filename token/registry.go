package token

import (
	"context"
	"encoding/json"
	"fmt"

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
}

func NewRegistry(rpcClient *rpc.Client) *Registry {
	return &Registry{
		rpcClient: rpcClient,
		names:     map[string]string{},
		store:     map[string]*RegistredToken{},
	}
}

func (r *Registry) GetToken(address *solana.PublicKey) *RegistredToken {
	return r.store[address.String()]
}

func (r *Registry) GetTokens() (out []*RegistredToken) {
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
	if err := r.loadNames(); err != nil {
		return fmt.Errorf("loading: name: %w", err)
	}

	address := "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	accounts, err := r.rpcClient.GetProgramAccounts(
		context.Background(),
		solana.MustPublicKeyFromBase58(address),
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
		r.store[aPub] = &RegistredToken{
			Address: a.Pubkey,
			Mint:    mint,
			Symbol:  r.names[aPub],
		}
	}
	return nil
}
