// Copyright 2021 dfuse Platform Inc.
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

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/logging"
	nodeManager "github.com/streamingfast/node-manager"
	nodeMindreaderStdinApp "github.com/streamingfast/node-manager/app/node_mindreader_stdin"
	"github.com/streamingfast/node-manager/metrics"
	"github.com/streamingfast/node-manager/mindreader"
	"github.com/streamingfast/sf-solana/codec"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	appLogger := zap.NewNop()
	logging.RegisterLogger("github.com/streamingfast/sf-ethereum/mindreader-node-stdin", appLogger)

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "mindreader-node-stdin",
		Title:       "Mindreader Node (stdin)",
		Description: "Blocks reading node, unmanaged, reads deep mind from standard input",
		MetricsID:   "mindreader-node-stdin",
		Logger:      launcher.NewLoggingDef("github.com/streamingfast/sf-ethereum/mindreader-node-stdin$", []zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel}),
		RegisterFlags: func(cmd *cobra.Command) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			sfDataDir := runtime.AbsDataDir
			archiveStoreURL := MustReplaceDataDir(sfDataDir, viper.GetString("common-oneblock-store-url"))
			mergeArchiveStoreURL := MustReplaceDataDir(sfDataDir, viper.GetString("common-blocks-store-url"))

			consoleReaderFactory := func(lines chan string) (mindreader.ConsolerReader, error) {
				r, err := codec.NewConsoleReader(lines, viper.GetString("mindreader-node-deepmind-batch-files-path"))
				if err != nil {
					return nil, fmt.Errorf("initiating console reader: %w", err)
				}
				return r, nil
			}

			consoleReaderBlockTransformer := func(obj interface{}) (*bstream.Block, error) {
				blk, ok := obj.(*pbcodec.Block)
				if !ok {
					return nil, fmt.Errorf("expected *pbcodec.Block, got %T", obj)
				}

				return codec.BlockFromProto(blk)
			}

			metricID := "mindreader-node-stdin"
			headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(metricID)
			headBlockNumber := metrics.NewHeadBlockNumber(metricID)
			metricsAndReadinessManager := nodeManager.NewMetricsAndReadinessManager(headBlockTimeDrift, headBlockNumber, viper.GetDuration("mindreader-node-readiness-max-latency"))

			blockmetaAddr := viper.GetString("common-blockmeta-addr")
			tracker := runtime.Tracker.Clone()
			tracker.AddGetter(bstream.NetworkLIBTarget, bstream.NetworkLIBBlockRefGetter(blockmetaAddr))
			return nodeMindreaderStdinApp.New(&nodeMindreaderStdinApp.Config{
				GRPCAddr:                   viper.GetString("mindreader-node-grpc-listen-addr"),
				ArchiveStoreURL:            archiveStoreURL,
				MergeArchiveStoreURL:       mergeArchiveStoreURL,
				BatchMode:                  viper.GetBool("mindreader-node-merge-and-store-directly"),
				MergeThresholdBlockAge:     viper.GetDuration("mindreader-node-merge-threshold-block-age"),
				MindReadBlocksChanCapacity: viper.GetInt("mindreader-node-blocks-chan-capacity"),
				StartBlockNum:              viper.GetUint64("mindreader-node-start-block-num"),
				StopBlockNum:               viper.GetUint64("mindreader-node-stop-block-num"),
				// Not used must kill
				//DiscardAfterStopBlock:        viper.GetBool("mindreader-geth-node-discard-after-stop-num"),
				FailOnNonContinuousBlocks:    viper.GetBool("mindreader-node-fail-on-non-contiguous-block"),
				WorkingDir:                   MustReplaceDataDir(sfDataDir, viper.GetString("mindreader-node-working-dir")),
				WaitUploadCompleteOnShutdown: viper.GetDuration("mindreader-node-wait-upload-complete-on-shutdown"),
				OneblockSuffix:               viper.GetString("mindreader-node-oneblock-suffix"),
			}, &nodeMindreaderStdinApp.Modules{
				ConsoleReaderFactory:       consoleReaderFactory,
				ConsoleReaderTransformer:   consoleReaderBlockTransformer,
				MetricsAndReadinessManager: metricsAndReadinessManager,
				Tracker:                    tracker,
			}, appLogger), nil
		},
	})
}
