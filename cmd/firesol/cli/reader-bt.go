package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/streamingfast/dstore"
	"go.uber.org/zap"

	"github.com/spf13/viper"
	"github.com/streamingfast/bstream/blockstream"
	nodeManagerSol "github.com/streamingfast/firehose-solana/node-manager"
	nodeManager "github.com/streamingfast/node-manager"
	nodeManagerApp "github.com/streamingfast/node-manager/app/node_manager2"
	"github.com/streamingfast/node-manager/metrics"
	"github.com/streamingfast/node-manager/operator"
	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"
	pbheadinfo "github.com/streamingfast/pbgo/sf/headinfo/v1"
	"google.golang.org/grpc"

	"github.com/spf13/cobra"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/logging"
)

func init() {
	appLogger, appTracer := logging.PackageLogger("reader", fmt.Sprintf("github.com/streamingfast/firehose-solana/reader-bt"))
	nodeLogger, _ := logging.PackageLogger("node", fmt.Sprintf("github.com/streamingfast/firehose-solana/reader/bigtable"))
	launcher.RegisterApp(zlog, &launcher.AppDef{
		ID:          "reader-bt",
		Title:       "Reader Node (bt)",
		Description: "Solana bigtable reader node with built-in operational manager",
		RegisterFlags: func(cmd *cobra.Command) error {
			registerCommonNodeFlags(cmd, "reader-bt")
			cmd.Flags().String("reader-bt-project-id", "", "Solana Bigtable Project ID")
			cmd.Flags().String("reader-bt-instance-id", "", "Solana Bigtable Instance ID")
			cmd.Flags().String("reader-bt-path", "firesol", "command that will be launched by the node manager")
			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			app := "reader-bt"
			dataDir := runtime.AbsDataDir

			mergedBlocksStoreURL, oneBlockStoreURL, _, err := getCommonStoresURLs(runtime.AbsDataDir)
			if err != nil {
				return nil, fmt.Errorf("unable to get common block store: %w", err)
			}

			mergedBlocksStore, err := dstore.NewDBinStore(mergedBlocksStoreURL)
			if err != nil {
				return nil, fmt.Errorf("unable to create merged blocks store at path %q: %w", mergedBlocksStoreURL, err)
			}

			providedStartBlock := viper.GetUint64("common-first-streamable-block")
			btProjectID := viper.GetString(app + "-project-id")
			btInstanceID := viper.GetString(app + "-instance-id")
			startBlockNum := resolveStartBlockNum(context.Background(), providedStartBlock, mergedBlocksStore)
			stopBlockNum := uint64(0)
			(*appLogger).Info("resolving start block",
				zap.Uint64("provided_start_block_num", providedStartBlock),
				zap.String("merged_block_store_url", mergedBlocksStoreURL),
				zap.Uint64("resolved_start_block_num", startBlockNum),
			)

			// resolve start block based on store

			(*appLogger).Info("configuring bigtable readers for syncing",
				zap.String("bt_project_id", btProjectID),
				zap.String("bt_instance_id", btInstanceID),
				zap.Uint64("start_block_num", startBlockNum),
				zap.Uint64("stop_block_num", stopBlockNum),
			)

			args := []string{
				"tools",
				"bt",
				"blocks",
				"--bt-project",
				btProjectID,
				"--bt-instance",
				btInstanceID,
				fmt.Sprintf("%d", startBlockNum),
				fmt.Sprintf("%d", stopBlockNum),
				"--firehose-enabled",
			}

			headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(app)
			headBlockNumber := metrics.NewHeadBlockNumber(app)
			appReadiness := metrics.NewAppReadiness(app)
			metricsAndReadinessManager := nodeManager.NewMetricsAndReadinessManager(
				headBlockTimeDrift,
				headBlockNumber,
				appReadiness,
				viper.GetDuration(app+"-readiness-max-latency"),
			)

			nodePath := viper.GetString(app + "-path")
			superviser, err := nodeManagerSol.NewSuperviser(
				nodePath,
				appLogger,
				nodeLogger,
				&nodeManagerSol.Options{
					BinaryPath:          nodePath,
					Arguments:           args,
					DataDirPath:         MustReplaceDataDir(dataDir, viper.GetString(app+"-data-dir")),
					DebugFirehoseLogs:   viper.GetBool(app + "-debug-firehose-logs"),
					LogToZap:            viper.GetBool(app + "-log-to-zap"),
					HeadBlockUpdateFunc: metricsAndReadinessManager.UpdateHeadBlock,
				})
			if err != nil {
				return nil, fmt.Errorf("unable to create chain superviser: %w", err)
			}

			chainOperator, err := operator.New(
				appLogger,
				superviser,
				metricsAndReadinessManager,
				&operator.Options{
					ShutdownDelay:              viper.GetDuration(app + "-shutdown-delay"),
					EnableSupervisorMonitoring: true,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("unable to create chain operator: %w", err)
			}

			zlog.Info("preparing reader plugin")
			blockStreamServer := blockstream.NewUnmanagedServer(blockstream.ServerOptionWithLogger(appLogger))
			workingDir := MustReplaceDataDir(dataDir, viper.GetString(app+"-working-dir"))
			blocksChanCapacity := viper.GetInt(app + "-blocks-chan-capacity")
			oneBlockFileSuffix := viper.GetString(app + "-one-block-suffix")
			consoleReaderFactory := getBigtableConsoleReaderFactory(appLogger)
			readerPlugin, err := getReaderLogPlugin(
				blockStreamServer,
				oneBlockStoreURL,
				workingDir,
				startBlockNum,
				stopBlockNum,
				blocksChanCapacity,
				oneBlockFileSuffix,
				chainOperator.Shutdown,
				consoleReaderFactory,
				metricsAndReadinessManager,
				appLogger,
				appTracer,
			)
			if err != nil {
				return nil, fmt.Errorf("new reader plugin: %w", err)
			}

			superviser.RegisterLogPlugin(readerPlugin)
			startupDelay := viper.GetDuration(app + "-startup-delay")
			return nodeManagerApp.New(&nodeManagerApp.Config{
				HTTPAddr:     viper.GetString(app + "-http-listen-addr"),
				GRPCAddr:     viper.GetString(app + "-grpc-listen-addr"),
				StartupDelay: startupDelay,
			}, &nodeManagerApp.Modules{
				Operator:                   chainOperator,
				MindreaderPlugin:           readerPlugin,
				MetricsAndReadinessManager: metricsAndReadinessManager,
				RegisterGRPCService: func(server grpc.ServiceRegistrar) error {
					pbheadinfo.RegisterHeadInfoServer(server, blockStreamServer)
					pbbstream.RegisterBlockStreamServer(server, blockStreamServer)
					return nil
				},
			}, appLogger), nil
		},
	})
}

func resolveStartBlockNum(ctx context.Context, start uint64, store dstore.Store) uint64 {
	errDone := errors.New("done")

	var seenStart *uint64
	var seenEnd *uint64

	err := store.WalkFrom(ctx, "", fmt.Sprintf("%010d", start), func(filename string) error {
		num, err := strconv.ParseUint(filename, 10, 64)
		if err != nil {
			return err
		}

		if num < start { // user has decided to start its merger in the 'future'
			return nil
		}

		if num == start {
			seenStart = &num
			return nil
		}

		// num > start
		if seenStart == nil {
			seenEnd = &num
			return errDone // first block after a hole
		}

		// increment by 100
		if num == *seenStart+100 {
			seenStart = &num
			return nil
		}

		return errDone
	})

	if err != nil && !errors.Is(err, errDone) {
		zlog.Error("got error walking store", zap.Error(err))
		return start
	}

	switch {
	case seenStart == nil:
		return start // nothing was found
	default:
		return *seenStart + 100
	}

}
