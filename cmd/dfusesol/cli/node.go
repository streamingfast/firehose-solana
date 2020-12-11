package cli

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	nodeManagerSol "github.com/dfuse-io/dfuse-solana/node-manager"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/dfuse-io/logging"
	nodeManager "github.com/dfuse-io/node-manager"
	nodeManagerApp "github.com/dfuse-io/node-manager/app/node_manager"
	"github.com/dfuse-io/node-manager/metrics"
	"github.com/dfuse-io/node-manager/operator"
	"github.com/dfuse-io/node-manager/profiler"
	solana "github.com/dfuse-io/solana-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var managerAPIPortByKind = map[string]string{
	"miner":   MinerNodeHTTPServingAddr,
	"peering": PeeringNodeHTTPServingAddr,
}

var rpcPortByKind = map[string]string{
	"miner":   MinerNodeRPCPort,
	"peering": PeeringNodeRPCPort,
}

var gossipPortByKind = map[string]string{
	"miner":   MinerNodeGossipPort,
	"peering": PeeringNodeGossipPort,
}

var p2pPortStartByKind = map[string]string{
	"miner":   MinerNodeP2PPortStart,
	"peering": PeeringNodeP2PPortStart,
}

var p2pPortEndByKind = map[string]string{
	"miner":   MinerNodeP2PPortEnd,
	"peering": PeeringNodeP2PPortEnd,
}

// RegisterSolanaNodeApp is an helper function that registers a given Solana node app. The `kind` value determines
// how the app is configured initial as well as being used to register flags, loggers, metrics, etc.
//
// Supported `kind`:
// - miner
// - peering
func RegisterSolanaNodeApp(kind string) {
	if kind != "miner" && kind != "peering" {
		panic(fmt.Errorf("invalid kind value, must be either 'miner' or 'peering', got %q", kind))
	}

	app := fmt.Sprintf("%s-node", kind)
	appLogger := zap.NewNop()
	nodeLogger := zap.NewNop()

	logging.Register(fmt.Sprintf("github.com/dfuse-io/dfuse-solana/%s", app), &appLogger)
	logging.Register(fmt.Sprintf("github.com/dfuse-io/dfuse-solana/%s/node", app), &nodeLogger)

	launcher.RegisterApp(&launcher.AppDef{
		ID:          app,
		Title:       fmt.Sprintf("Solana Node (%s)", kind),
		Description: fmt.Sprintf("Solana %s node with built-in operational manager", kind),
		MetricsID:   app,
		Logger: launcher.NewLoggingDef(
			fmt.Sprintf("github.com/dfuse-io/dfuse-solana/%s.*", app),
			[]zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel},
		),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String(app+"-network", "development", "Which network this node refers to, 'development' ")
			cmd.Flags().String(app+"-config-dir", "./"+kind, "Directory for config files")
			cmd.Flags().String(app+"-data-dir", fmt.Sprintf("{dfuse-data-dir}/%s/data", kind), "Directory for data (node blocks and state)")
			cmd.Flags().String(app+"-rpc-port", rpcPortByKind[kind], "HTTP listening port of Solana node, setting this to empty string disable RPC endpoint for the node")
			cmd.Flags().String(app+"-gossip-port", gossipPortByKind[kind], "TCP gossip listening port of Solana node")
			cmd.Flags().String(app+"-p2p-port-start", p2pPortStartByKind[kind], "P2P dynamic range start listening port of Solana node")
			cmd.Flags().String(app+"-p2p-port-end", p2pPortEndByKind[kind], "P2P dynamic range end of Solana node")
			cmd.Flags().String(app+"-manager-api-addr", managerAPIPortByKind[kind], "Solana node manager API address")
			cmd.Flags().Duration(app+"-readiness-max-latency", 30*time.Second, "The health endpoint '/healthz' will return an error until the head block time is within that duration to now")
			cmd.Flags().Duration(app+"-shutdown-delay", 0, "Delay before shutting manager when sigterm received")
			cmd.Flags().String(app+"-extra-arguments", "", "Extra arguments to be passed when executing superviser binary")
			cmd.Flags().String(app+"-bootstrap-data-url", "", "URL where to find bootstrapping data for this node, the URL must point to a `.tar.zst` archive containing the full data directory to bootstrap from")
			cmd.Flags().Bool(app+"-disable-profiler", true, "Disables the node manager profiler")
			cmd.Flags().Bool(app+"-log-to-zap", true, "Enable all node logs to transit into app's logger directly, when false, prints node logs directly to stdout")
			cmd.Flags().Bool(app+"-debug-deep-mind", false, "[DEV] Prints deep mind instrumentation logs to standard output, should be use for debugging purposes only")
			cmd.Flags().Bool(app+"-rpc-enable-debug-apis", false, "[DEV] Enable some of the Solana validator RPC APIs that can be used for debugging purposes")

			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			dataDir := mustReplaceDataDir(dfuseDataDir, viper.GetString(app+"-data-dir"))
			configDir, err := filepath.Abs(viper.GetString(app + "-config-dir"))
			if err != nil {
				return nil, fmt.Errorf("invalid config directory %q: %w", viper.GetString(app+"-config-dir"), err)
			}

			headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(app)
			headBlockNumber := metrics.NewHeadBlockNumber(app)

			var p *profiler.Profiler
			if !viper.GetBool(app + "-disable-profiler") {
				p = profiler.GetInstance(appLogger)
			}

			identityFile := filepath.Join(configDir, "identity.json")
			if !mustFileExists(identityFile) {
				return nil, fmt.Errorf("identity file %q does not exist but it should", identityFile)
			}

			voteAccountFile := identityFile
			if mustFileExists(filepath.Join(configDir, "vote-account.json")) {
				voteAccountFile = filepath.Join(configDir, "vote-account.json")
			}

			arguments := append([]string{
				"--identity", identityFile,
				"--vote-account", voteAccountFile,
				"--ledger", dataDir,
				"--gossip-port", viper.GetString(app + "-gossip-port"),
				"--dynamic-port-range", fmt.Sprintf("%s-%s", viper.GetString(app+"-p2p-port-start"), viper.GetString(app+"-p2p-port-end")),
				"--log", "-",
			})

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

			if kind == "miner" {
				arguments = append(arguments,
					"--init-complete-file", filepath.Join(dataDir, "init-complete.log"),
				)
			}

			network := viper.GetString(app + "-network")
			startupDelay := time.Duration(0)

			if kind == "peering" || kind == "mindreader" {
				appLogger.Info("configuring node for syncing", zap.String("network", network))
				// FIXME: Maybe some of those flags are good only for development networks ... not clear yet
				arguments = append(arguments,
					"--halt-on-trusted-validators-accounts-hash-mismatch",
					"--limit-ledger-size",
					"--no-untrusted-rpc",
					"--no-voting",
				)

				if network == "development" {
					appLogger.Info("configuring node for development syncing")
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
				} else if network == "custom" {
					appLogger.Info("configuring node for custom syncing, you are expected to provide the required arguments through the '" + app + "-extra-arguments' flag")
				} else {
					return nil, fmt.Errorf(`unkown network %q, valid networks are "development", "custom"`, network)
				}
			}

			if kind == "mindreader" {
				appLogger.Info("configuring node as a mindreader")
				arguments = append(arguments,
					"--no-snapshot-fetch",
				)
			}

			extraArguments := viper.GetString(app + "-extra-arguments")
			if extraArguments != "" {
				for _, arg := range strings.Split(extraArguments, " ") {
					arguments = append(arguments, arg)
				}
			}

			metricsAndReadinessManager := nodeManager.NewMetricsAndReadinessManager(
				headBlockTimeDrift,
				headBlockNumber,
				viper.GetDuration(app+"-readiness-max-latency"),
			)

			superviser, err := nodeManagerSol.NewSuperviser(appLogger, nodeLogger, &nodeManagerSol.Options{
				BinaryPath:          viper.GetString("global-validator-path"),
				Arguments:           arguments,
				DataDirPath:         mustReplaceDataDir(dfuseDataDir, viper.GetString(app+"-data-dir")),
				DebugDeepMind:       viper.GetBool(app + "-debug-deep-mind"),
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
					BootstrapDataURL:           viper.GetString(app + "-bootstrap-data-url"),
					Profiler:                   p,
				},
			)

			if err != nil {
				return nil, fmt.Errorf("unable to create chain operator: %w", err)
			}

			return nodeManagerApp.New(&nodeManagerApp.Config{
				ManagerAPIAddress: viper.GetString(app + "-manager-api-addr"),
				StartupDelay:      startupDelay,
			}, &nodeManagerApp.Modules{
				Operator:                   chainOperator,
				MetricsAndReadinessManager: metricsAndReadinessManager,
			}, appLogger), nil
		},
	})
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
	configValue := mustReplaceDataDir(runtime.AbsDataDir, viper.GetString(kind+"-node-data-dir"))
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
