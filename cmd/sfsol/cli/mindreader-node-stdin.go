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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/logging"
	nodeManager "github.com/streamingfast/node-manager"
	nodeMindreaderStdinApp "github.com/streamingfast/node-manager/app/node_mindreader_stdin"
	"github.com/streamingfast/node-manager/metrics"
)

func init() {
	appLogger, appTracer := logging.PackageLogger("mindreader-node-stdin", "github.com/streamingfast/sf-ethereum/mindreader-node-stdin")

	launcher.RegisterApp(zlog, &launcher.AppDef{
		ID:          "mindreader-node-stdin",
		Title:       "Mindreader Node (stdin)",
		Description: "Blocks reading node, unmanaged, reads deep mind from standard input",
		RegisterFlags: func(cmd *cobra.Command) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			sfDataDir := runtime.AbsDataDir
			archiveStoreURL := MustReplaceDataDir(sfDataDir, viper.GetString("common-oneblock-store-url"))
			mergeArchiveStoreURL := MustReplaceDataDir(sfDataDir, viper.GetString("common-blocks-store-url"))

			consoleReaderFactory := getConsoleReaderFactory(
				appLogger,
				viper.GetString("mindreader-node-deepmind-batch-files-path"),
				viper.GetBool("mindreader-node-purge-account-data"),
			)
			metricID := "mindreader-node-stdin"
			headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(metricID)
			headBlockNumber := metrics.NewHeadBlockNumber(metricID)
			metricsAndReadinessManager := nodeManager.NewMetricsAndReadinessManager(headBlockTimeDrift, headBlockNumber, viper.GetDuration("mindreader-node-readiness-max-latency"))

			return nodeMindreaderStdinApp.New(&nodeMindreaderStdinApp.Config{
				GRPCAddr:                     viper.GetString("mindreader-node-grpc-listen-addr"),
				ArchiveStoreURL:              archiveStoreURL,
				MergeArchiveStoreURL:         mergeArchiveStoreURL,
				MergeThresholdBlockAge:       viper.GetString("mindreader-node-merge-threshold-block-age"),
				MindReadBlocksChanCapacity:   viper.GetInt("mindreader-node-blocks-chan-capacity"),
				StartBlockNum:                viper.GetUint64("mindreader-node-start-block-num"),
				StopBlockNum:                 viper.GetUint64("mindreader-node-stop-block-num"),
				WorkingDir:                   MustReplaceDataDir(sfDataDir, viper.GetString("mindreader-node-working-dir")),
				WaitUploadCompleteOnShutdown: viper.GetDuration("mindreader-node-wait-upload-complete-on-shutdown"),
				OneblockSuffix:               viper.GetString("mindreader-node-oneblock-suffix"),
				LogToZap:                     viper.GetBool("mindreader-node-log-to-zap"),
				DebugDeepMind:                viper.GetBool("mindreader-node-debug-deep-mind"),
			}, &nodeMindreaderStdinApp.Modules{
				ConsoleReaderFactory:       consoleReaderFactory,
				MetricsAndReadinessManager: metricsAndReadinessManager,
			}, appLogger, appTracer), nil
		},
	})
}
