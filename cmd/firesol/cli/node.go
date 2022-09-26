package cli

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lorenzosaino/go-sysctl"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream/blockstream"
	"github.com/streamingfast/dlauncher/launcher"
	nodeManagerSol "github.com/streamingfast/firehose-solana/node-manager"
	"github.com/streamingfast/logging"
	nodeManager "github.com/streamingfast/node-manager"
	nodeManagerApp "github.com/streamingfast/node-manager/app/node_manager2"
	"github.com/streamingfast/node-manager/metrics"
	"github.com/streamingfast/node-manager/operator"
	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"
	pbheadinfo "github.com/streamingfast/pbgo/sf/headinfo/v1"
	"github.com/streamingfast/solana-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var httpListenAddrByKind = map[string]string{
	"miner":   MinerNodeHTTPServingAddr,
	"reader":  ReaderNodeHTTPServingAddr,
	"peering": PeeringNodeHTTPServingAddr,
}

var rpcPortByKind = map[string]string{
	"miner":   MinerNodeRPCPort,
	"reader":  ReaderNodeRPCPort,
	"peering": PeeringNodeRPCPort,
}

var gossipPortByKind = map[string]string{
	"miner":   MinerNodeGossipPort,
	"reader":  ReaderNodeGossipPort,
	"peering": PeeringNodeGossipPort,
}

var p2pPortStartByKind = map[string]string{
	"miner":   MinerNodeP2PPortStart,
	"reader":  ReaderNodeP2PPortStart,
	"peering": PeeringNodeP2PPortStart,
}

var p2pPortEndByKind = map[string]string{
	"miner":   MinerNodeP2PPortEnd,
	"reader":  ReaderNodeP2PPortEnd,
	"peering": PeeringNodeP2PPortEnd,
}

// flags common to reader and regular node
// app is expected to either be 'reader-node' or 'reader-bt'
func registerCommonNodeFlags(cmd *cobra.Command, app string) {
	cmd.Flags().Duration(app+"-readiness-max-latency", 30*time.Second, "The health endpoint '/healthz' will return an error until the head block time is within that duration to now")
	cmd.Flags().String(app+"-data-dir", fmt.Sprintf("{data-dir}/%s/data", "reader"), "Directory for data (node blocks and state)")
	cmd.Flags().Bool(app+"-debug-firehose-logs", false, "[DEV] Prints Firehose logs to standard output, should be use for debugging purposes only")
	cmd.Flags().Bool(app+"-log-to-zap", true, "Enable all node logs to transit into app's logger directly, when false, prints node logs directly to stdout")
	cmd.Flags().Duration(app+"-shutdown-delay", 0, "Delay before shutting manager when sigterm received")
	cmd.Flags().String(app+"-working-dir", "{data-dir}/reader/work", "Path where reader will stores its files")
	cmd.Flags().Int(app+"-blocks-chan-capacity", 100, "Capacity of the channel holding blocks read by the reader. Process will shutdown superviser/geth if the channel gets over 90% of that capacity to prevent horrible consequences. Raise this number when processing tiny blocks very quickly")
	cmd.Flags().String(app+"-one-block-suffix", "", "If non-empty, the oneblock files will be appended with that suffix, so that readers can each write their file for a given block instead of competing for writes.")
	cmd.Flags().Duration(app+"-startup-delay", 0, "[DEV] wait time before launching")

}

func nodeFactoryFunc(app string, appLogger *zap.Logger, appTracer logging.Tracer, nodeLogger *zap.Logger) func(*launcher.Runtime) (launcher.App, error) {
	return func(runtime *launcher.Runtime) (launcher.App, error) {
		fmt.Println("YOU ARE HEREHEREHEREHEREHEREHEREHEREHEREHERE")
		if err := setupNodeSysctl(appLogger); err != nil {
			return nil, fmt.Errorf("systcl configuration for %s failed: %w", app, err)
		}

		dataDir := runtime.AbsDataDir
		args := []string{}

		rpcPort := viper.GetString(app + "-rpc-port")
		if rpcPort != "" {
			args = append(args, "--rpc-port", rpcPort)
			if viper.GetBool(app + "-rpc-enable-debug-apis") {
				args = append(args,
					"--enable-rpc-exit",
					"--enable-rpc-set-log-filter",
					// FIXME: Not really a debug stuff, but usefull to have there for easier developer work
					"--enable-rpc-transaction-history",
				)
			}
		}

		network := viper.GetString(app + "-network")
		startupDelay := viper.GetDuration(app + "-startup-delay")
		(*appLogger).Info("configuring node for syncing", zap.String("network", network))
		args = append(args, "--limit-ledger-size")
		if network == "development" {
			(*appLogger).Info("configuring node for development syncing")
			// FIXME: What a bummer, connection refused on cluster endpoint simply terminates the process!
			//        It means that we will need to implement something in the manager to track those kind
			//        of error and restart the manager manually!
			//
			//        For now in development, let 15s for miner app to properly start.
			startupDelay = 5 * time.Second

			minerPublicKey, err := readPublicKeyFromConfigFile("miner", "identity.json")
			if err != nil {
				return nil, fmt.Errorf("unable to read miner public key: %w", err)
			}

			minerGenesisHash, err := readConfigFile("miner", "genesis.hash")
			if err != nil {
				return nil, fmt.Errorf("unable to read miner genesis hash: %w", err)
			}

			minerGenesisShred, err := readConfigFile("miner", "genesis.shred")
			if err != nil {
				return nil, fmt.Errorf("unable to read miner genesis shred: %w", err)
			}

			args = append(args,
				"--entrypoint", "127.0.0.1:"+viper.GetString("miner-node-gossip-port"),
				"--trusted-validator", minerPublicKey.String(),

				// FIXME: In development, how could we actually read this data from somewhere? When bootstrap data is available, we
				//        could actually read from it. Otherwise, the init phase would have generated something, what do we do
				//        in this case? Maybe always generate .hash and .shred file just like in battlefield ...
				"--expected-genesis-hash", minerGenesisHash,
				"--expected-shred-version", minerGenesisShred,
			)
		} else if network == "mainnet-beta" {
			args = append(args)

		} else if network == "testnet" {
			args = append(args,
				"--entrypoint", "entrypoint.testnet.solana.com:8001",
				"--trusted-validator", "5D1fNXzvv5NjV1ysLjirC4WY92RNsVH18vjmcszZd8on",
				"--trusted-validator", "ta1Uvfb7W5BRPrdGnhP9RmeCGKzBySGM1hTE4rBRy6T",
				"--trusted-validator", "Ft5fbkqNa76vnsjYNwjDZUXoTWpP7VYm3mtsaQckQADN",
				"--trusted-validator", "9QxCLckBiJc783jnMvXZubK4wH86Eqqvashtrwvcsgkv",
				"--expected-genesis-hash", "4uhcVJyU9pJkvQyS88uRDiswHXSCkY3zQawwpjk2NsNY",
			)
		} else if network == "devnet" {
			args = append(args,
				"--entrypoint", "entrypoint.devnet.solana.com:8001",
				"--trusted-validator", "dv1LfzJvDF7S1fBKpFgKoKXK5yoSosmkAdfbxBo1GqJ",
				"--expected-genesis-hash", "EtWTRABZaYq6iMfeYKouRu166VU2xqa1wcaWoxPkrZBG",
			)
		} else if network == "custom" {
			(*appLogger).Info("configuring node for custom syncing, you are expected to provide the required arguments through the '" + app + "-arguments' flag")
		} else {
			return nil, fmt.Errorf(`unkown network %q, valid networks are "development", "mainnet-beta", "testnet", "devnet", "custom"`, network)
		}

		if providedArgument := viper.GetString(app + "-arguments"); providedArgument != "" {
			if strings.HasPrefix(providedArgument, "+") {
				(*appLogger).Info("appending provided arguments to default", zap.String("provided_arguments", providedArgument))
				args = append(args, strings.Split(strings.TrimLeft(providedArgument, "+"), " ")...)
			} else {
				(*appLogger).Info("overriding default arguments with provided arguments", zap.String("provided_arguments", providedArgument))
				args = strings.Split(providedArgument, " ")
			}
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

		superviser, err := nodeManagerSol.NewSuperviser(
			appLogger,
			nodeLogger,
			&nodeManagerSol.Options{
				BinaryPath:          viper.GetString("global-validator-path"),
				Arguments:           args,
				DataDirPath:         MustReplaceDataDir(dataDir, viper.GetString(app+"-data-dir")),
				DebugFirehoseLogs:   viper.GetBool(app + "-debug-firehose-logs"),
				LogToZap:            viper.GetBool(app + "-log-to-zap"),
				HeadBlockUpdateFunc: metricsAndReadinessManager.UpdateHeadBlock,
			})
		if err != nil {
			return nil, fmt.Errorf("unable to create chain superviser: %w", err)
		}
		var bootstrapper operator.Bootstrapper
		bootstrapDataURL := viper.GetString(app + "-bootstrap-data-url")

		if bootstrapDataURL != "" {
			bootstrapper = nodeManagerSol.NewBootstrapper(bootstrapDataURL, dataDir, appLogger)
		}

		chainOperator, err := operator.New(
			appLogger,
			superviser,
			metricsAndReadinessManager,
			&operator.Options{
				ShutdownDelay:              viper.GetDuration(app + "-shutdown-delay"),
				EnableSupervisorMonitoring: true,
				Bootstrapper:               bootstrapper,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("unable to create chain operator: %w", err)
		}

		zlog.Info("preparing reader plugin")
		_, oneBlockStoreURL, _, err := getCommonStoresURLs(runtime.AbsDataDir)
		if err != nil {
			return nil, fmt.Errorf("unable to get common block store: %w", err)
		}

		blockStreamServer := blockstream.NewUnmanagedServer(blockstream.ServerOptionWithLogger(appLogger))
		workingDir := MustReplaceDataDir(dataDir, viper.GetString(app+"-working-dir"))
		batchStartBlockNum := viper.GetUint64(app + "-start-block-num")
		batchStopBlockNum := viper.GetUint64(app + "-stop-block-num")
		blocksChanCapacity := viper.GetInt(app + "-blocks-chan-capacity")
		oneBlockFileSuffix := viper.GetString(app + "-one-block-suffix")
		batchFilePath := viper.GetString("reader-node-firehose-batch-files-path")
		purgeAccountChanges := viper.GetBool("reader-node-purge-account-data")
		consoleReaderFactory := getConsoleReaderFactory(appLogger, batchFilePath, purgeAccountChanges)
		readerPlugin, err := getReaderLogPlugin(
			blockStreamServer,
			oneBlockStoreURL,
			workingDir,
			batchStartBlockNum,
			batchStopBlockNum,
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
	}
}

func readPublicKeyFromConfigFile(kind string, file string) (out solana.PublicKey, err error) {
	privateKeyFile, err := getConfigFilePath(kind, file)
	if err != nil {
		return out, fmt.Errorf("config file: %w", err)
	}

	privateKey, err := solana.PrivateKeyFromSolanaKeygenFile(privateKeyFile)
	if err != nil {
		return out, fmt.Errorf("read private key file %q: %w", privateKeyFile, err)
	}

	return privateKey.PublicKey(), nil
}

func getConfigFilePath(kind string, file string) (string, error) {
	configValue := viper.GetString(kind + "-node-config-dir")
	configDir, err := filepath.Abs(configValue)
	if err != nil {
		return "", fmt.Errorf("invalid config directory %q: %w", configValue, err)
	}

	return filepath.Join(configDir, file), nil
}

func getDataFilePath(runtime *launcher.Runtime, kind string, file string) (string, error) {
	configValue := MustReplaceDataDir(runtime.AbsDataDir, viper.GetString(kind+"-node-data-dir"))
	dataDir, err := filepath.Abs(configValue)
	if err != nil {
		return "", fmt.Errorf("invalid data directory %q: %w", configValue, err)
	}

	return filepath.Join(dataDir, file), nil
}

func readConfigFile(kind string, file string) (string, error) {
	configFile, err := getConfigFilePath(kind, file)
	if err != nil {
		return "", fmt.Errorf("config file: %w", err)
	}

	out, err := ioutil.ReadFile(configFile)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	if len(out) == 0 {
		return "", nil
	}

	return string(out), nil
}

func getExtraArguments(kind string) (out []string) {
	extraArguments := viper.GetString(kind + "-node-extra-arguments")
	if extraArguments != "" {
		for _, arg := range strings.Split(extraArguments, " ") {
			out = append(out, arg)
		}
	}
	return
}

func setupNodeSysctl(logger *zap.Logger) error {
	if runtime.GOOS == "darwin" {
		logger.Debug("skipping sysctl vm.max_map_count checks for Darwin OSs (Mac OS X)")
		return nil
	}

	out, err := sysctl.Get("vm.max_map_count")
	if err != nil {
		return fmt.Errorf("can't retrieve value for vm.max_map_count sysctl: %w", err)
	}

	val, err := strconv.Atoi(out)
	if err != nil {
		return fmt.Errorf("can't convert value %q of vm.max_map_count: %w", out, err)
	}

	if val < 500000 {
		return fmt.Errorf("vm.max_map_count too low, set it to at least 500000 (sysctl -w vm.max_map_count=500000)")
	}

	return nil
}
