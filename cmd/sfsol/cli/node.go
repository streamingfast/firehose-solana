package cli

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lorenzosaino/go-sysctl"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream/blockstream"
	"github.com/streamingfast/dlauncher/launcher"
	nodeManager "github.com/streamingfast/node-manager"
	nodeManagerApp "github.com/streamingfast/node-manager/app/node_manager2"
	"github.com/streamingfast/node-manager/metrics"
	"github.com/streamingfast/node-manager/mindreader"
	"github.com/streamingfast/node-manager/operator"
	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"
	pbheadinfo "github.com/streamingfast/pbgo/sf/headinfo/v1"
	nodeManagerSol "github.com/streamingfast/sf-solana/node-manager"
	"github.com/streamingfast/solana-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var httpListenAddrByKind = map[string]string{
	"miner":      MinerNodeHTTPServingAddr,
	"mindreader": MindreaderNodeHTTPServingAddr,
	"peering":    PeeringNodeHTTPServingAddr,
}

var rpcPortByKind = map[string]string{
	"miner":      MinerNodeRPCPort,
	"mindreader": MindreaderNodeRPCPort,
	"peering":    PeeringNodeRPCPort,
}

var gossipPortByKind = map[string]string{
	"miner":      MinerNodeGossipPort,
	"mindreader": MindreaderNodeGossipPort,
	"peering":    PeeringNodeGossipPort,
}

var p2pPortStartByKind = map[string]string{
	"miner":      MinerNodeP2PPortStart,
	"mindreader": MindreaderNodeP2PPortStart,
	"peering":    PeeringNodeP2PPortStart,
}

var p2pPortEndByKind = map[string]string{
	"miner":      MinerNodeP2PPortEnd,
	"mindreader": MindreaderNodeP2PPortEnd,
	"peering":    PeeringNodeP2PPortEnd,
}

func nodeFactoryFunc(app, kind string, appLogger, nodeLogger *zap.Logger) func(*launcher.Runtime) (launcher.App, error) {
	return func(runtime *launcher.Runtime) (launcher.App, error) {
		if err := setupNodeSysctl(appLogger); err != nil {
			return nil, fmt.Errorf("systcl configuration for %s failed: %w", app, err)
		}

		sfDataDir := runtime.AbsDataDir
		dataDir := MustReplaceDataDir(sfDataDir, viper.GetString(app+"-data-dir"))

		headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(app)
		headBlockNumber := metrics.NewHeadBlockNumber(app)

		arguments := append([]string{})

		rpcPort := viper.GetString(app + "-rpc-port")
		if rpcPort != "" {
			arguments = append(arguments,
				"--rpc-port", rpcPort,
			)

			if viper.GetBool(app + "-rpc-enable-debug-apis") {
				arguments = append(arguments,
					"--enable-rpc-exit",
					"--enable-rpc-set-log-filter",
					// FIXME: Not really a debug stuff, but usefull to have there for easier developer work
					"--enable-rpc-transaction-history",
				)
			}
		}

		network := viper.GetString(app + "-network")
		startupDelay := viper.GetDuration(app + "-startup-delay")
		extraArguments := getExtraArguments(kind)

		if kind == "mindreader" {
			(*appLogger).Info("configuring node for syncing", zap.String("network", network))

			arguments = append(arguments, "--limit-ledger-size")
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

				arguments = append(arguments,
					"--entrypoint", "127.0.0.1:"+viper.GetString("miner-node-gossip-port"),
					"--trusted-validator", minerPublicKey.String(),

					// FIXME: In development, how could we actually read this data from somewhere? When bootstrap data is available, we
					//        could actually read from it. Otherwise, the init phase would have generated something, what do we do
					//        in this case? Maybe always generate .hash and .shred file just like in battlefield ...
					"--expected-genesis-hash", minerGenesisHash,
					"--expected-shred-version", minerGenesisShred,
				)
			} else if network == "mainnet-beta" {
				arguments = append(arguments)

			} else if network == "testnet" {
				arguments = append(arguments,
					"--entrypoint", "entrypoint.testnet.solana.com:8001",
					"--trusted-validator", "5D1fNXzvv5NjV1ysLjirC4WY92RNsVH18vjmcszZd8on",
					"--trusted-validator", "ta1Uvfb7W5BRPrdGnhP9RmeCGKzBySGM1hTE4rBRy6T",
					"--trusted-validator", "Ft5fbkqNa76vnsjYNwjDZUXoTWpP7VYm3mtsaQckQADN",
					"--trusted-validator", "9QxCLckBiJc783jnMvXZubK4wH86Eqqvashtrwvcsgkv",
					"--expected-genesis-hash", "4uhcVJyU9pJkvQyS88uRDiswHXSCkY3zQawwpjk2NsNY",
				)
			} else if network == "devnet" {
				arguments = append(arguments,
					"--entrypoint", "entrypoint.devnet.solana.com:8001",
					"--trusted-validator", "dv1LfzJvDF7S1fBKpFgKoKXK5yoSosmkAdfbxBo1GqJ",
					"--expected-genesis-hash", "EtWTRABZaYq6iMfeYKouRu166VU2xqa1wcaWoxPkrZBG",
				)
			} else if network == "custom" {
				(*appLogger).Info("configuring node for custom syncing, you are expected to provide the required arguments through the '" + app + "-extra-arguments' flag")
			} else {
				return nil, fmt.Errorf(`unkown network %q, valid networks are "development", "mainnet-beta", "testnet", "devnet", "custom"`, network)
			}
		}

		if len(extraArguments) > 0 {
			arguments = append(arguments, extraArguments...)
		}

		metricsAndReadinessManager := nodeManager.NewMetricsAndReadinessManager(
			headBlockTimeDrift,
			headBlockNumber,
			viper.GetDuration(app+"-readiness-max-latency"),
		)

		superviser, err := nodeManagerSol.NewSuperviser(
			appLogger,
			nodeLogger,
			&nodeManagerSol.Options{
				BinaryPath:          viper.GetString("global-validator-path"),
				Arguments:           arguments,
				DataDirPath:         MustReplaceDataDir(sfDataDir, viper.GetString(app+"-data-dir")),
				DebugDeepMind:       viper.GetBool(app + "-debug-deep-mind"),
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
		mergedBlocksStoreURL := MustReplaceDataDir(sfDataDir, viper.GetString("common-blocks-store-url"))

		var mindreaderPlugin *mindreader.MindReaderPlugin
		var registerServices func(server *grpc.Server) error

		if kind == "mindreader" {
			zlog.Info("preparing mindreader plugin")
			blockStreamServer := blockstream.NewUnmanagedServer(blockstream.ServerOptionWithLogger(appLogger))
			oneBlockStoreURL := MustReplaceDataDir(sfDataDir, viper.GetString("common-oneblock-store-url"))

			mergeAndStoreDirectly := viper.GetBool(app + "-merge-and-store-directly")
			mergeThresholdBlockAge := viper.GetDuration(app + "-merge-threshold-block-age")
			workingDir := MustReplaceDataDir(sfDataDir, viper.GetString(app+"-working-dir"))
			blockDataWorkingDir := MustReplaceDataDir(sfDataDir, viper.GetString(app+"-block-data-working-dir"))
			batchStartBlockNum := viper.GetUint64("mindreader-node-start-block-num")
			batchStopBlockNum := viper.GetUint64("mindreader-node-stop-block-num")
			blocksChanCapacity := viper.GetInt("mindreader-node-blocks-chan-capacity")
			failOnNonContiguousBlock := viper.GetBool("mindreader-node-fail-on-non-contiguous-block")
			waitTimeForUploadOnShutdown := viper.GetDuration("mindreader-node-wait-upload-complete-on-shutdown")
			oneBlockFileSuffix := viper.GetString("mindreader-node-oneblock-suffix")
			batchFilePath := viper.GetString("mindreader-node-deepmind-batch-files-path")
			purgeAccountChanges := viper.GetBool("mindreader-node-purge-account-data")
			tracker := runtime.Tracker.Clone()

			mindreaderPlugin, err = getMindreaderLogPlugin(
				blockStreamServer,
				oneBlockStoreURL,
				mergedBlocksStoreURL,
				mergeAndStoreDirectly,
				mergeThresholdBlockAge,
				workingDir,
				blockDataWorkingDir,
				batchStartBlockNum,
				batchStopBlockNum,
				blocksChanCapacity,
				failOnNonContiguousBlock,
				waitTimeForUploadOnShutdown,
				oneBlockFileSuffix,
				chainOperator.Shutdown,
				metricsAndReadinessManager,
				tracker,
				appLogger,
				batchFilePath,
				purgeAccountChanges,
			)
			if err != nil {
				return nil, fmt.Errorf("new mindreader plugin: %w", err)
			}

			registerServices = func(server *grpc.Server) error {
				pbheadinfo.RegisterHeadInfoServer(server, blockStreamServer)
				pbbstream.RegisterBlockStreamServer(server, blockStreamServer)

				return nil
			}

			superviser.RegisterLogPlugin(mindreaderPlugin)
		}

		return nodeManagerApp.New(&nodeManagerApp.Config{
			HTTPAddr:     viper.GetString(app + "-http-listen-addr"),
			GRPCAddr:     viper.GetString(app + "-grpc-listen-addr"),
			StartupDelay: startupDelay,
		}, &nodeManagerApp.Modules{
			Operator:                   chainOperator,
			MindreaderPlugin:           mindreaderPlugin,
			MetricsAndReadinessManager: metricsAndReadinessManager,
			RegisterGRPCService:        registerServices,
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

func hasExtraArgument(arguments []string, flag string) bool {
	for _, argument := range arguments {
		parts := strings.Split(argument, "=")
		if parts[0] == flag {
			return true
		}
	}

	return false
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
