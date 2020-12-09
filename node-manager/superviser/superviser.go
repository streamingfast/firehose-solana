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

package superviser

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dfuse-io/solana-go/rpc"

	"github.com/ShinyTrinkets/overseer"
	nodeManager "github.com/dfuse-io/node-manager"
	logplugin "github.com/dfuse-io/node-manager/log_plugin"
	"github.com/dfuse-io/node-manager/metrics"
	"github.com/dfuse-io/node-manager/superviser"
	"go.uber.org/zap"
)

type NodeosSuperviser struct {
	*superviser.Superviser
	name string

	logger  *zap.Logger
	options *SuperviserOption
	client  *rpc.Client
}
type SuperviserOption struct {
	BinaryPath          string
	SolanaBinaryOptions SolanaBinaryOptions
	LogToZap            bool
}

type SolanaBinaryOptions interface {
	toArgs() []string
	rpcPort() int
}
type LocalSolanaBinaryOptions struct {
	ValidatorIdentityFilePath       string
	ValidatorVoteAccountFilePath    string
	LedgerDirectoryPath             string
	InitialisationCompletedFilePath string
	RPCPort                         int
	GossipPort                      int
	RPCFaucetAddress                string
	TrustedValidator                []string
}

func (o *LocalSolanaBinaryOptions) toArgs() []string {
	var args []string

	args = append(args, "--identity="+o.ValidatorIdentityFilePath)
	args = append(args, "--vote-account="+o.ValidatorVoteAccountFilePath)
	args = append(args, "--ledger="+o.LedgerDirectoryPath)
	args = append(args, fmt.Sprintf("--gossip-port=%d", o.GossipPort))
	args = append(args, fmt.Sprintf("--rpc-port=%d", o.RPCPort))
	args = append(args, "--rpc-faucet-address="+o.RPCFaucetAddress)
	args = append(args, "--init-complete-file="+o.InitialisationCompletedFilePath)

	args = append(args, "--log=-")
	args = append(args, "--enable-rpc-exit")
	args = append(args, "--enable-rpc-transaction-history")
	args = append(args, "--snapshot-compression=none")
	args = append(args, "--require-tower")

	return args
}

func (o *LocalSolanaBinaryOptions) rpcPort() int {
	return o.RPCPort
}

type MainSolanaBinaryOptions struct {
	RPCPort             int
	LedgerDirectoryPath string
}

func (o *MainSolanaBinaryOptions) toArgs() []string {

	var args []string

	args = append(args, "--ledger="+o.LedgerDirectoryPath)
	args = append(args, "--no-port-check")
	//args = append(args, "--no-snapshot-fetch")
	args = append(args, "--no-voting")
	args = append(args, "--trusted-validator=7Np41oeYqPefeNQEHSv1UDhYrehxin3NStELsSKCT4K2")
	args = append(args, "--trusted-validator=GdnSyH3YtwcxFvQrVVJMm1JhTS4QVX7MFsX56uJLUfiZ")
	args = append(args, "--trusted-validator=DE1bawNcRJB9rVm3buyMVfr8mBEoyyu73NBovf2oXJsJ")
	args = append(args, "--trusted-validator=CakcnaRDHka2gXyfbEd2d3xsvkJkqsLw2akB3zsN1D2S")
	args = append(args, "--no-untrusted-rpc")
	args = append(args, "--rpc-port=8899")
	args = append(args, "--private-rpc")
	args = append(args, "--dynamic-port-range=8000-8010")
	args = append(args, "--entrypoint=mainnet-beta.solana.com:8001")
	args = append(args, "--expected-genesis-hash=5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d")
	args = append(args, "--log=-")
	//args = append(args, "--cuda")
	//args = append(args, "--enable-rpc-transaction-history")
	//args = append(args, "--wal-recovery-mode skip_any_corrupted_record")
	//args = append(args, "--limit-ledger-size")

	return args

}

func (o *MainSolanaBinaryOptions) rpcPort() int {
	return o.RPCPort
}

func NewSuperviser(debugDeepMind bool, options *SuperviserOption, logger *zap.Logger) (*NodeosSuperviser, error) {
	// Ensure process manager line buffer is large enough (50 MiB) for our Deep Mind instrumentation outputting lot's of text.
	overseer.DEFAULT_LINE_BUFFER_SIZE = 50 * 1024 * 1024
	client := rpc.NewClient(fmt.Sprintf("http://127.0.0.1:%d", options.SolanaBinaryOptions.rpcPort()))
	s := &NodeosSuperviser{
		// The arguments field is actually `nil` because arguments are re-computed upon each start
		Superviser: superviser.New(logger, options.BinaryPath, nil),
		options:    options,
		logger:     logger,
		client:     client,
	}

	//todo: this is use for the wait for reader.
	//s.RegisterLogPlugin(logplugin.LogPluginFunc(s.analyzeLogLineForStateChange))

	if options.LogToZap {
		s.RegisterLogPlugin(logplugin.NewToZapLogPlugin(debugDeepMind, logger))
	} else {
		s.RegisterLogPlugin(logplugin.NewToConsoleLogPlugin(debugDeepMind))
	}

	return s, nil
}

//func (s *NodeosSuperviser) GetCommand() string {
//	return s.options.BinaryPath + " " + s.options.toArgs()
//}

func (s *NodeosSuperviser) Start(options ...nodeManager.StartOption) error {
	s.Logger.Info("updating nodeos arguments before starting binary")
	s.Superviser.Arguments = s.options.SolanaBinaryOptions.toArgs()

	s.logger.Info("Starting solana-validator", zap.String("command", s.GetCommand()))
	err := s.Superviser.Start(options...)
	if err != nil {
		return err
	}

	return nil
}

func (s *NodeosSuperviser) IsRunning() bool {
	isRunning := s.Superviser.IsRunning()
	isRunningMetricsValue := float64(0)
	if isRunning {
		isRunningMetricsValue = float64(1)
	}

	metrics.NodeosCurrentStatus.SetFloat64(isRunningMetricsValue)

	return isRunning
}

//this is for logging purpose only
func (s *NodeosSuperviser) GetCommand() string {
	return s.options.BinaryPath + " " + strings.Join(s.options.SolanaBinaryOptions.toArgs(), " ")
}

func (s *NodeosSuperviser) HasData() bool {
	return true
}

func (s *NodeosSuperviser) ServerID() (string, error) {
	return os.Hostname()
}

func (s *NodeosSuperviser) LastSeenBlockNum() uint64 {
	r, err := s.client.GetSlot(context.Background(), rpc.CommitmentRecent)
	if err != nil {
		zlog.Error("Failed to get last seen slot from rpc client. returning 0", zap.Error(err))
	}

	return uint64(r)
}

func (s *NodeosSuperviser) GetName() string {
	return "solana-validator"
}
