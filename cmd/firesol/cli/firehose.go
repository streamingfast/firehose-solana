package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/streamingfast/bstream/transform"
	"github.com/streamingfast/dstore"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	dauthAuthenticator "github.com/streamingfast/dauth/authenticator"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/dmetering"
	"github.com/streamingfast/dmetrics"
	firehoseApp "github.com/streamingfast/firehose/app/firehose"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/substreams/client"
	substreamsService "github.com/streamingfast/substreams/service"
	"go.uber.org/zap"
)

var metricset = dmetrics.NewSet()
var headBlockNumMetric = metricset.NewHeadBlockNumber("firehose")
var headTimeDriftmetric = metricset.NewHeadTimeDrift("firehose")

func init() {
	appLogger := zap.NewNop()
	appLogger, _ = logging.PackageLogger("firehose", "github.com/streamingfast/firehose-solana/firehose")

	launcher.RegisterApp(zlog, &launcher.AppDef{
		ID:          "firehose",
		Title:       "Block Firehose",
		Description: "Provides on-demand filtered blocks, depends on common-merged-blocks-store-url and common-live-blocks-addr",
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("firehose-grpc-listen-addr", FirehoseGRPCServingAddr, "Address on which the Firehose will listen")

			cmd.Flags().Bool("substreams-enabled", false, "Whether to enable substreams")
			cmd.Flags().Bool("substreams-partial-mode-enabled", false, "Whether to enable partial stores generation support on this instance (usually for internal deployments only)")
			cmd.Flags().String("substreams-state-store-url", "{data-dir}/localdata", "where substreams state data are stored")
			cmd.Flags().Uint64("substreams-stores-save-interval", uint64(1_000), "Interval in blocks at which to save store snapshots")     // fixme
			cmd.Flags().Uint64("substreams-output-cache-save-interval", uint64(100), "Interval in blocks at which to save store snapshots") // fixme
			cmd.Flags().String("substreams-client-endpoint", "", "Firehose endpoint for substreams client.  if left empty, will default to this current local Firehose.")
			cmd.Flags().String("substreams-client-jwt", "", "jwt for substreams client authentication")
			cmd.Flags().Bool("substreams-client-insecure", false, "substreams client in insecure mode")
			cmd.Flags().Bool("substreams-client-plaintext", true, "substreams client in plaintext mode")
			cmd.Flags().Int("substreams-sub-request-parallel-jobs", 5, "substreams subrequest parallel jobs for the scheduler")
			cmd.Flags().Int("substreams-sub-request-block-range-size", 1000, "substreams subrequest block range size value for the scheduler")

			return nil
		},

		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dataDir := runtime.AbsDataDir

			blockstreamAddr := viper.GetString("common-live-blocks-addr")

			// FIXME: That should be a shared dependencies across `EOSIO on StreamingFast`
			authenticator, err := dauthAuthenticator.New(viper.GetString("common-auth-plugin"))
			if err != nil {
				return nil, fmt.Errorf("unable to initialize dauth: %w", err)
			}

			// FIXME: That should be a shared dependencies across `EOSIO on StreamingFast` it will avoid the need to call `dmetering.SetDefaultMeter`
			metering, err := dmetering.New(viper.GetString("common-metering-plugin"))
			if err != nil {
				return nil, fmt.Errorf("unable to initialize dmetering: %w", err)
			}
			dmetering.SetDefaultMeter(metering)

			mergedBlocksStoreURL, oneBlockStoreURL, forkedBlocksStoreURL, err := getCommonStoresURLs(runtime.AbsDataDir)
			if err != nil {
				return nil, fmt.Errorf("failed to get common store URL: %w", err)
			}

			shutdownSignalDelay := viper.GetDuration("common-system-shutdown-signal-delay")
			grcpShutdownGracePeriod := time.Duration(0)
			if shutdownSignalDelay.Seconds() > 5 {
				grcpShutdownGracePeriod = shutdownSignalDelay - (5 * time.Second)
			}

			var registerServiceExt firehoseApp.RegisterServiceExtensionFunc
			if viper.GetBool("substreams-enabled") {

				stateStore, err := dstore.NewStore(MustReplaceDataDir(dataDir, viper.GetString("substreams-state-store-url")), "", "", true)
				if err != nil {
					return nil, fmt.Errorf("setting up state store for data: %w", err)
				}

				opts := []substreamsService.Option{
					substreamsService.WithStoresSaveInterval(viper.GetUint64("substreams-stores-save-interval")),
					substreamsService.WithOutCacheSaveInterval(viper.GetUint64("substreams-output-cache-save-interval")),
				}

				if viper.GetBool("substreams-partial-mode-enabled") {
					opts = append(opts, substreamsService.WithPartialMode())
				}

				ssClientInsecure := viper.GetBool("substreams-client-insecure")
				ssClientPlaintext := viper.GetBool("substreams-client-plaintext")
				if ssClientInsecure && ssClientPlaintext {
					return nil, fmt.Errorf("cannot set both substreams-client-insecure and substreams-client-plaintext")
				}

				endpoint := viper.GetString("substreams-client-endpoint")
				if endpoint == "" {
					endpoint = viper.GetString("firehose-grpc-listen-addr")
				}

				substreamsClientConfig := client.NewSubstreamsClientConfig(
					endpoint,
					os.ExpandEnv(viper.GetString("substreams-client-jwt")),
					viper.GetBool("substreams-client-insecure"),
					viper.GetBool("substreams-client-plaintext"),
				)

				sss, err := substreamsService.New(
					stateStore,
					"sf.solana.type.v1.Block",
					viper.GetInt("substreams-sub-request-parallel-jobs"),
					viper.GetInt("substreams-sub-request-block-range-size"),
					substreamsClientConfig,
					opts...,
				)
				if err != nil {
					return nil, fmt.Errorf("creating substreams service: %w", err)
				}

				registerServiceExt = sss.Register
			}

			registry := transform.NewRegistry()

			return firehoseApp.New(appLogger, &firehoseApp.Config{
				MergedBlocksStoreURL:    mergedBlocksStoreURL,
				OneBlocksStoreURL:       oneBlockStoreURL,
				ForkedBlocksStoreURL:    forkedBlocksStoreURL,
				BlockStreamAddr:         blockstreamAddr,
				GRPCListenAddr:          viper.GetString("firehose-grpc-listen-addr"),
				GRPCShutdownGracePeriod: grcpShutdownGracePeriod,
				ServiceDiscoveryURL:     nil,
			}, &firehoseApp.Modules{
				Authenticator:            authenticator,
				HeadTimeDriftMetric:      headTimeDriftmetric,
				HeadBlockNumberMetric:    headBlockNumMetric,
				TransformRegistry:        registry,
				RegisterServiceExtension: registerServiceExt,
			}), nil
		},
	})
}
