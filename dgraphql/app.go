// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dgraphql

import (
	"context"
	"fmt"

	drateLimiter "github.com/dfuse-io/dauth/ratelimiter"
	solResolver "github.com/dfuse-io/dfuse-solana/dgraphql/resolvers"
	"github.com/dfuse-io/dfuse-solana/dgraphql/trade"
	"github.com/dfuse-io/dfuse-solana/token"
	"github.com/dfuse-io/dfuse-solana/transaction"
	"github.com/dfuse-io/dgraphql"
	dgraphqlApp "github.com/dfuse-io/dgraphql/app/dgraphql"
	"github.com/dfuse-io/solana-go/rpc"
	"go.uber.org/zap"
)

type Config struct {
	dgraphqlApp.Config
	RatelimiterPlugin string
	RPCEndpointAddr   string
	RPCWSEndpointAddr string
	SlotOffset        uint64
}

func NewApp(config *Config) (*dgraphqlApp.App, error) {
	zlog.Info("new dgraphql app", zap.Reflect("config", config))

	dgraphqlBaseConfig := config.Config
	return dgraphqlApp.New(&dgraphqlBaseConfig, &dgraphqlApp.Modules{
		PredefinedGraphqlExamples: GraphqlExamples(),
		SchemaFactory:             &SchemaFactory{config: config},
	}), nil
}

type SchemaFactory struct {
	config *Config
}

func (f *SchemaFactory) Schemas() (*dgraphql.Schemas, error) {
	// FIXME: The context should be provided by the caller of `Schemas` which should be tied to the app lifecycle
	ctx := context.Background()

	rateLimiter, err := drateLimiter.New(f.config.RatelimiterPlugin)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize rate limiter: %w", err)
	}

	rpcClient := rpc.NewClient(f.config.RPCEndpointAddr)
	tradeManager := trade.NewManager()
	trxStream := transaction.NewStream(rpcClient, f.config.RPCWSEndpointAddr, tradeManager, f.config.SlotOffset)
	tokenRegistry := token.NewRegistry(rpcClient, f.config.RPCWSEndpointAddr)

	err = trxStream.Launch(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to launch trx stream: %w", err)
	}

	zlog.Info("configuring resolver and parsing schemas")
	resolver, err := solResolver.NewRoot(
		rpcClient,
		f.config.RPCWSEndpointAddr,
		tradeManager,
		tokenRegistry,
		rateLimiter,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create root resolver: %w", err)
	}

	schemas, err := dgraphql.NewSchemas(resolver)
	if err != nil {
		return nil, fmt.Errorf("unable to parse schema: %w", err)
	}

	return schemas, nil
}