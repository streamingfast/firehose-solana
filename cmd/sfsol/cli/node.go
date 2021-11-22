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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/blockstream"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/logging"
	nodeManager "github.com/streamingfast/node-manager"
	nodeManagerApp "github.com/streamingfast/node-manager/app/node_manager2"
	"github.com/streamingfast/node-manager/metrics"
	"github.com/streamingfast/node-manager/mindreader"
	"github.com/streamingfast/node-manager/operator"
	pbbstream "github.com/streamingfast/pbgo/dfuse/bstream/v1"
	pbheadinfo "github.com/streamingfast/pbgo/dfuse/headinfo/v1"
	"github.com/streamingfast/sf-solana/codec"
	nodeManagerSol "github.com/streamingfast/sf-solana/node-manager"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"github.com/streamingfast/solana-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// RegisterSolanaNodeApp is an helper function that registers a given Solana node app. The `kind` value determines
// how the app is configured initial as well as being used to register flags, loggers, metrics, etc.
//
// Supported `kind`:
// - miner
// - peering
func RegisterSolanaNodeApp(kind string, extraFlagRegistration func(cmd *cobra.Command) error) {
	if kind != "miner" && kind != "mindreader" && kind != "peering" {
		panic(fmt.Errorf("invalid kind value, must be either 'miner', 'mindreader' or 'peering', got %q", kind))
	}

	app := fmt.Sprintf("%s-node", kind)
	appLogger := zap.NewNop()
	nodeLogger := zap.NewNop()

	logging.Register(fmt.Sprintf("github.com/streamingfast/sf-solana/%s", app), &appLogger)
	logging.Register(fmt.Sprintf("github.com/streamingfast/sf-solana/%s/node", app), &nodeLogger)

	launcher.RegisterApp(&launcher.AppDef{
		ID:          app,
		Title:       fmt.Sprintf("Solana Node (%s)", kind),
		Description: fmt.Sprintf("Solana %s node with built-in operational manager", kind),
		MetricsID:   app,
		Logger: launcher.NewLoggingDef(
			fmt.Sprintf("github.com/streamingfast/sf-solana/%s.*", app),
			[]zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel},
		),
		RegisterFlags: func(cmd *cobra.Command) error {
			registerCommonNodeFlags(cmd, app, kind)
			extraFlagRegistration(cmd)
			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) error {
			return nil
		},
		FactoryFunc: nodeFactoryFunc(app, kind, appLogger, nodeLogger),
	})
}

func nodeFactoryFunc(app, kind string, appLogger, nodeLogger *zap.Logger) func(*launcher.Runtime) (launcher.App, error) {
	return func(runtime *launcher.Runtime) (launcher.App, error) {
		if err := setupNodeSysctl(appLogger); err != nil {
			return nil, fmt.Errorf("systcl configuration for %s failed: %w", app, err)
		}

		sfDataDir := runtime.AbsDataDir

		dataDir := mustReplaceDataDir(sfDataDir, viper.GetString(app+"-data-dir"))
		configDir, err := filepath.Abs(viper.GetString(app + "-config-dir"))
		if err != nil {
			return nil, fmt.Errorf("invalid config directory %q: %w", viper.GetString(app+"-config-dir"), err)
		}

		headBlockTimeDrift := metrics.NewHeadBlockTimeDrift(app)
		headBlockNumber := metrics.NewHeadBlockNumber(app)

		arguments := append([]string{
			"--ledger", dataDir,
			//"--gossip-port", viper.GetString(app + "-gossip-port"),
			//"--dynamic-port-range", fmt.Sprintf("%s-%s", viper.GetString(app+"-p2p-port-start"), viper.GetString(app+"-p2p-port-end")),
			"--log", "-",
			//"--no-snapshot-fetch",
			//"--no-genesis-fetch",
		})
		if app == "miner" {
			identityFile := filepath.Join(configDir, "identity.json")
			if !mustFileExists(identityFile) {
				return nil, fmt.Errorf("identity file %q does not exist but it should", identityFile)
			}

			voteAccountFile := identityFile
			if mustFileExists(filepath.Join(configDir, "vote-account.json")) {
				voteAccountFile = filepath.Join(configDir, "vote-account.json")
			}

			arguments = append(arguments,
				"--identity", identityFile,
				"--vote-account", voteAccountFile,
			)
		}

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
		startupDelay := viper.GetDuration(app + "-startup-delay")
		extraArguments := getExtraArguments(kind)

		if kind == "peering" || kind == "mindreader" {
			appLogger.Info("configuring node for syncing", zap.String("network", network))

			arguments = append(arguments, "--limit-ledger-size")
			if kind == "peering" {
				arguments = append(arguments, "400000000")
			}

			// FIXME: Maybe some of those flags are good only for development networks ... not clear yet
			//arguments = append(arguments,
			//	"--halt-on-trusted-validators-accounts-hash-mismatch",
			//	"--no-untrusted-rpc",
			//	"--no-voting",
			//	"--private-rpc",
			//	"--wal-recovery-mode", "skip_any_corrupted_record",
			//)

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
			} else if network == "mainnet-beta" {
				arguments = append(arguments) //"--entrypoint", "mainnet-beta.solana.com:8001",
				//"--trusted-validator", "2xte5CBkCBEBLNviyAfvSaTkMy6tvg99Cy3XJj9EJJs2",
				//"--trusted-validator", "3Ec6j5SkCj51PgH2fBUcXc4ptrwi6i5WpnCxZBak3cX3",
				//"--trusted-validator", "3KNGMiXwhy2CAWVNpLoUt25sNngFnX1mZpaiEeVccBA6",
				//"--trusted-validator", "3LboiLyZ3U1556ZBnNi9384C8Gz1LxFmzRnAojumnCJB",
				//"--trusted-validator", "3RbsAuNknCTXuLyqmasnvYRpQg3MfWZ5N7WTi7ZGqdms",
				//"--trusted-validator", "4TJZp9Ho82FrcRcBQes5oD52Y3QYeCxkpqWmjxmySQFY",
				//"--trusted-validator", "5i6FL92EgjMmtFRogXeB7TaCYYAwd7kTQ9abKc1RTRMf",
				//"--trusted-validator", "6GRLDLiAtx8ZjYgQgPo7UsYeJ9g1pLX5j3HK97tFmtXb",
				//"--trusted-validator", "6cgsK8ph5tNUCiKG5WXLMZFX1CoL4jzuVouTPBwPC8fk",
				//"--trusted-validator", "7Np41oeYqPefeNQEHSv1UDhYrehxin3NStELsSKCT4K2",
				//"--trusted-validator", "7XSCAfoJ11zrQxonjbGZHLUL8tqpF7yhkxiieLds9mdH",
				//"--trusted-validator", "8DM7JdDPShN4TFrWTwokvWmnCDqCy1D6VVLzSMDKri5V",
				//"--trusted-validator", "8DXdM93UpEfqXezv1QTPhuA7Rci8MZujhsXQHoAsx5cN",
				//"--trusted-validator", "9EBnt7k6Z4mRCiFMCN1kT8joN3SWHCuhQrW1a8M8mYPM",
				//"--trusted-validator", "9hdNfC1DKGXxyqbNRSsStiPYoUREoRWZAEhmiqPw93yP",
				//"--trusted-validator", "9rVx9wo6d3Akaq9YBw4sFuwc9oFGtzs8GsTfkHE7EH6t",
				//"--trusted-validator", "AUa3iN7h4c3oSrtP5pmbRcXJv8QSo4HGHPqXT4WnHDnp",
				//"--trusted-validator", "AaDBW2BYPC1cpCM6bYf5hN9MCNFz79fMHbK7VLXwrW5x",
				//"--trusted-validator", "AqGAaaACTDNGrVNVoiyCGiMZe8pcM1YjGUcUdVwgUtud",
				//"--trusted-validator", "BAbRkBYDhThcR8rn7wYtPYSxDnUCfRYx5dAjsuianuA6",
				//"--trusted-validator", "Bb4BP3EvsPyBuqSAABx7KmYAp3mRqAZUYN1vChWsbjDc",
				//"--trusted-validator", "CVAAQGA8GBzKi4kLdmpDuJnpkSik6PMWSvRk3RDds9K8",
				//"--trusted-validator", "CakcnaRDHka2gXyfbEd2d3xsvkJkqsLw2akB3zsN1D2S",
				//"--trusted-validator", "DE1bawNcRJB9rVm3buyMVfr8mBEoyyu73NBovf2oXJsJ",
				//"--trusted-validator", "DE37cgN2bGR26a1yQPPY42CozC1wXwXnTXDyyURHRsm7",
				//"--trusted-validator", "F3LudCbGqu4DMqjduLq5WE2g3USYcjmVK3WR8KeNBhWz",
				//"--trusted-validator", "FVsjR8faKFZSisBatLNVo5bSH1jvHz3JvneVbyVTiV9K",
				//"--trusted-validator", "GdnSyH3YtwcxFvQrVVJMm1JhTS4QVX7MFsX56uJLUfiZ",
				//"--trusted-validator", "GosJ8GHbSUunTQPY5xEyjhY2Eg5a9qSuPhNC4Ctztr7y",
				//"--trusted-validator", "HoMBSLMokd6BUVDT4iGw21Tnxvp2G49MApewzGJr4rfe",
				//"--trusted-validator", "HzrEstnLfzsijhaD6z5frkSE2vWZEH5EUfn3bU9swo1f",
				//"--trusted-validator", "HzvGtvXFzMeJwNYcUu5pw8yyRxF2tLEvDSSFsAEBcBK2",
				//"--trusted-validator", "J4B32eD2PmwCZyre5SWQS1jCU4NkiGGYLNapg9f8Dkqo",
				//"--trusted-validator", "ba2eZEU27TqR1MB9WUPJ2F7dcTrNsgdx38tBg53GexZ",
				//"--trusted-validator", "ba3zMkMp87HZg27Z7EDEkxE48zcKgJ59weFYtrKadY7",
				//"--trusted-validator", "ba5rfuZ37gxhrLcsgA5fzCg8BvSQcTERPqY14Qffa3J",
				//"--trusted-validator", "tEBPZWSAdpzQoVzWBFD2qVGmZ7vB3Mh1Jq4tGZBx5eA",

				//"--expected-shred-version", "13490",
				//"--expected-genesis-hash", "5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d",

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
				appLogger.Info("configuring node for custom syncing, you are expected to provide the required arguments through the '" + app + "-extra-arguments' flag")
			} else {
				return nil, fmt.Errorf(`unkown network %q, valid networks are "development", "mainnet-beta", "testnet", "devnet", "custom"`, network)
			}
		}

		if viper.GetDuration(app+"-auto-snapshot-period") != 0 && !hasExtraArgument(extraArguments, "--snapshot-interval-slots") {
			return nil, fmt.Errorf("extra argument --snapshot-interval-slots XYZ required if you pass in --%s-auto-snapshot-period", app)
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
				BinaryPath: viper.GetString("global-validator-path"),
				Arguments:  arguments,
				// BinaryPath:          "/bin/bash",
				// Arguments:           []string{"-c", `cat /tmp/mama.txt /home/abourget/build/solana/validator/dmlog.log; sleep 3600`},
				DataDirPath:         mustReplaceDataDir(sfDataDir, viper.GetString(app+"-data-dir")),
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

				//RestoreSnapshotName:     viper.GetString(app + "-restore-snapshot-name"),
				//NumberOfSnapshotsToKeep: viper.GetInt(app + "-number-of-snapshots-to-keep"),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("unable to create chain operator: %w", err)
		}
		mergedBlocksStoreURL := mustReplaceDataDir(sfDataDir, viper.GetString("common-blocks-store-url"))
		snapshotStoreURL := mustReplaceDataDir(sfDataDir, viper.GetString("common-snapshots-store-url"))
		if snapshotStoreURL != "" {
			localSnapshotDir := viper.GetString(app + "-local-snapshot-folder")
			genesisURL := viper.GetString(app + "-genesis-url")
			snapshotStore, err := dstore.NewSimpleStore(snapshotStoreURL)
			if err != nil {
				return nil, fmt.Errorf("unable to create snapshot store from url %q: %w", snapshotStoreURL, err)
			}

			mergedBlocksStore, err := dstore.NewDBinStore(mergedBlocksStoreURL)
			if err != nil {
				return nil, fmt.Errorf("setting up merged blocks store: %w", err)
			}

			snapshotter := nodeManagerSol.NewSnapshotter(localSnapshotDir, genesisURL, snapshotStore, mergedBlocksStore)
			chainOperator.RegisterBackupModule("snapshotter", snapshotter)

			// TODO check me
			chainOperator.RegisterBackupSchedule(&operator.BackupSchedule{
				TimeBetweenRuns:       viper.GetDuration(app + "-auto-snapshot-period"),
				RequiredHostnameMatch: "",
				BackuperName:          "snapshotter",
			})
		}

		var mindreaderPlugin *mindreader.MindReaderPlugin
		var registerServices func(server *grpc.Server) error

		if kind == "mindreader" {
			blockStreamServer := blockstream.NewUnmanagedServer(blockstream.ServerOptionWithLogger(appLogger))
			oneBlockStoreURL := mustReplaceDataDir(sfDataDir, viper.GetString("common-oneblock-store-url"))
			blockDataStoreURL := mustReplaceDataDir(sfDataDir, viper.GetString("common-block-data-store-url"))
			mergeAndStoreDirectly := viper.GetBool(app + "-merge-and-store-directly")
			mergeThresholdBlockAge := viper.GetDuration(app + "-merge-threshold-block-age")
			workingDir := mustReplaceDataDir(sfDataDir, viper.GetString(app+"-working-dir"))
			blockDataWorkingDir := mustReplaceDataDir(sfDataDir, viper.GetString(app+"-block-data-working-dir"))
			batchStartBlockNum := viper.GetUint64("mindreader-node-start-block-num")
			batchStopBlockNum := viper.GetUint64("mindreader-node-stop-block-num")
			blocksChanCapacity := viper.GetInt("mindreader-node-blocks-chan-capacity")
			failOnNonContiguousBlock := viper.GetBool("mindreader-node-fail-on-non-contiguous-block")
			waitTimeForUploadOnShutdown := viper.GetDuration("mindreader-node-wait-upload-complete-on-shutdown")
			oneBlockFileSuffix := viper.GetString("mindreader-node-oneblock-suffix")
			tracker := runtime.Tracker.Clone()

			mindreaderPlugin, err := getMindreaderLogPlugin(
				blockStreamServer,
				oneBlockStoreURL,
				blockDataStoreURL,
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

func consoleReaderBlockTransformerWithArchive(archiver *nodeManagerSol.BlockDataArchiver, obj interface{}) (*bstream.Block, error) {
	block, ok := obj.(*pbcodec.Block)
	zlog.Debug("transforming block", zap.Uint64("slot_num", block.Number))
	if !ok {
		return nil, fmt.Errorf("expected *pbcodec.Block, got %T", obj)
	}

	fileName := blockDataFileName(block, "")
	block.AccountChangesFileRef = archiver.Store.ObjectURL(fileName)
	zlog.Debug("slot data file", zap.String("object_url", block.AccountChangesFileRef))

	accountChangesBundle := block.Split(true)
	go func() {
		err := archiver.StoreBlockData(accountChangesBundle, fileName)
		if err != nil {
			//todo: This is very bad
			panic(fmt.Errorf("storing block data: %w", err))
		}
		zlog.Debug("slot data store", zap.String("object_path", block.AccountChangesFileRef))

	}()

	bstreamBlock, err := codec.BlockFromProto(block)
	if err != nil {
		return nil, fmt.Errorf("block from proto: %w", err)
	}

	return bstreamBlock, nil
}

//duplicated code from node manager
func blockDataFileName(slot *pbcodec.Block, suffix string) string {
	t := slot.Time()
	blockTimeString := fmt.Sprintf("%s.%01d", t.Format("20060102T150405"), t.Nanosecond()/100000000)

	blockID := slot.ID()
	if len(blockID) > 8 {
		blockID = blockID[len(blockID)-8:]
	}

	previousID := slot.PreviousID()
	if len(previousID) > 8 {
		previousID = previousID[len(previousID)-8:]
	}

	suffixString := ""
	if suffix != "" {
		suffixString = fmt.Sprintf("-%s", suffix)
	}
	return fmt.Sprintf("%010d-%s-%s-%s%s", slot.Num(), blockTimeString, blockID, previousID, suffixString)
}

func registerCommonNodeFlags(cmd *cobra.Command, flagPrefix string, kind string) {
	cmd.Flags().String(flagPrefix+"-network", "development", "Which network this node refers to, 'development' ")
	cmd.Flags().String(flagPrefix+"-config-dir", "./"+kind, "Directory for config files")
	cmd.Flags().String(flagPrefix+"-data-dir", fmt.Sprintf("{sf-data-dir}/%s/data", kind), "Directory for data (node blocks and state)")
	cmd.Flags().String(flagPrefix+"-rpc-port", rpcPortByKind[kind], "HTTP listening port of Solana node, setting this to empty string disable RPC endpoint for the node")
	cmd.Flags().String(flagPrefix+"-gossip-port", gossipPortByKind[kind], "TCP gossip listening port of Solana node")
	cmd.Flags().String(flagPrefix+"-p2p-port-start", p2pPortStartByKind[kind], "P2P dynamic range start listening port of Solana node")
	cmd.Flags().String(flagPrefix+"-p2p-port-end", p2pPortEndByKind[kind], "P2P dynamic range end of Solana node")
	cmd.Flags().String(flagPrefix+"-http-listen-addr", httpListenAddrByKind[kind], "Solana node manager HTTP address when operational command can be send to control the node")
	cmd.Flags().Duration(flagPrefix+"-readiness-max-latency", 30*time.Second, "The health endpoint '/healthz' will return an error until the head block time is within that duration to now")
	cmd.Flags().Duration(flagPrefix+"-shutdown-delay", 0, "Delay before shutting manager when sigterm received")
	cmd.Flags().String(flagPrefix+"-extra-arguments", "", "Extra arguments to be passed when executing superviser binary")
	cmd.Flags().String(flagPrefix+"-bootstrap-data-url", "", "URL where to find bootstrapping data for this node, the URL must point to a `.tar.zst` archive containing the full data directory to bootstrap from")
	cmd.Flags().Bool(flagPrefix+"-log-to-zap", true, "Enable all node logs to transit into app's logger directly, when false, prints node logs directly to stdout")
	cmd.Flags().Bool(flagPrefix+"-rpc-enable-debug-apis", false, "[DEV] Enable some of the Solana validator RPC APIs that can be used for debugging purposes")
	cmd.Flags().Duration(flagPrefix+"-startup-delay", 0, "[DEV] wait time before launching")

	// We don't want to handle `backups` right now.. only `snapshot`.
	//cmd.Flags().String("node-manager-auto-restore-source", "snapshot", "Enables restore from the latest source. Can be either, 'snapshot' or 'backup'. Do not use 'backup' on single block producing node")
	cmd.Flags().String(flagPrefix+"-restore-snapshot-name", "", "If non-empty, the node will be restored from that snapshot when it starts.")
	cmd.Flags().Duration(flagPrefix+"-auto-snapshot-period", 0, "If non-zero, the node manager will check on disk at this period interval to see if the underlying node has produced a snapshot. Use in conjunction with --snapshot-interval-slots in the --"+flagPrefix+"-extra-arguments. Specify 1m, 2m...")
	cmd.Flags().String(flagPrefix+"-local-snapshot-folder", "", "where solana snapshots are stored by the node")
	cmd.Flags().Int(flagPrefix+"-number-of-snapshots-to-keep", 0, "if non-zero, after a successful snapshot, older snapshots will be deleted to only keep that number of recent snapshots")
	cmd.Flags().String(flagPrefix+"-genesis-url", "", "url to genesis.tar.bz2")
}
