package graphql

import (
	"context"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-solana/graphql/server"
	"github.com/dfuse-io/dfuse-solana/graphql/trade"
	"github.com/dfuse-io/dfuse-solana/token"
	"github.com/dfuse-io/dfuse-solana/transaction"
	"github.com/dfuse-io/shutter"
	"github.com/dfuse-io/solana-go/rpc"
	"go.uber.org/zap"
)

type Config struct {
	BlockStoreURL     string // GS path to read batch files from
	RPCURL            string
	HTTPListenAddress string
	RPCWSURL          string
	SlotOffset        uint64
}

type Modules struct {
}
type App struct {
	*shutter.Shutter
	config    *Config
	modules   *Modules
	ReadyFunc func()
	isReady   func() bool
}

func New(config *Config, modules *Modules) *App {
	return &App{
		Shutter:   shutter.New(),
		config:    config,
		modules:   modules,
		ReadyFunc: func() {},
	}
}

func (a *App) Run() error {
	zlog.Info("running graphql application", zap.Reflect("config", a.config))

	ctx := context.Background()

	rpcClient := rpc.NewClient(a.config.RPCURL)

	tradeManager := trade.NewManager()

	trxStream := transaction.NewStream(rpcClient, a.config.RPCWSURL, tradeManager, a.config.SlotOffset)

	tokenRegistry := token.NewRegistry(rpcClient, a.config.RPCWSURL)

	err := trxStream.Launch(ctx)
	derr.Check("launch trx stream", err)

	s := server.NewServer(
		a.config.HTTPListenAddress,
		rpcClient,
		tradeManager,
		a.config.RPCWSURL,
		tokenRegistry,
	)
	zlog.Info("serving ...")

	return s.Launch()
}
