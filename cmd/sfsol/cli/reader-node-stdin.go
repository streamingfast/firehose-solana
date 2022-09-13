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
	nodeReaderStdinApp "github.com/streamingfast/node-manager/app/node_mindreader_stdin"
	"github.com/streamingfast/node-manager/metrics"
)

func init() {
	appLogger, appTracer := logging.PackageLogger("reader-node-stdin", "github.com/streamingfast/sf-ethereum/mindreader-node-stdin")

	launcher.RegisterApp(zlog, &launcher.AppDef{
		ID:          "reader-node-stdin",
		Title:       "Reader Node (stdin)",
		Description: "Blocks reading node, unmanaged, reads firehose logs from standard input",
		RegisterFlags: func(cmd *cobra.Command) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			sfDataDir := runtime.AbsDataDir
			archiveStoreURL := MustReplaceDataDir(sfDataDir, viper.GetString("common-oneblock-store-url"))
			mergeArchiveStoreURL := MustReplaceDataDir(sfDataDir, viper.GetString("common-blocks-store-url"))

			consoleReaderFactory := getConsoleReaderFactory(
				appLogger,
				viper.GetString("reader-node-firehose-batch-files-path"),
				viper.GetBool("reader-node-purge-account-data"),
			)
			metricID := "reader-node-stdin"
			headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(metricID)
			headBlockNumber := metrics.NewHeadBlockNumber(metricID)
			metricsAndReadinessManager := nodeManager.NewMetricsAndReadinessManager(headBlockTimeDrift, headBlockNumber, viper.GetDuration("reader-node-readiness-max-latency"))

			return nodeReaderStdinApp.New(&nodeReaderStdinApp.Config{
				GRPCAddr:                     viper.GetString("reader-node-grpc-listen-addr"),
				ArchiveStoreURL:              archiveStoreURL,
				MergeArchiveStoreURL:         mergeArchiveStoreURL,
				MergeThresholdBlockAge:       viper.GetString("reader-node-merge-threshold-block-age"),
				MindReadBlocksChanCapacity:   viper.GetInt("reader-node-blocks-chan-capacity"),
				StartBlockNum:                viper.GetUint64("reader-node-start-block-num"),
				StopBlockNum:                 viper.GetUint64("reader-node-stop-block-num"),
				WorkingDir:                   MustReplaceDataDir(sfDataDir, viper.GetString("reader-node-working-dir")),
				WaitUploadCompleteOnShutdown: viper.GetDuration("reader-node-wait-upload-complete-on-shutdown"),
				OneblockSuffix:               viper.GetString("reader-node-oneblock-suffix"),
				LogToZap:                     viper.GetBool("reader-node-log-to-zap"),
				DebugDeepMind:                viper.GetBool("reader-node-debug-firehose-logs"),
			}, &nodeReaderStdinApp.Modules{
				ConsoleReaderFactory:       consoleReaderFactory,
				MetricsAndReadinessManager: metricsAndReadinessManager,
			}, appLogger, appTracer), nil
		},
	})
}
