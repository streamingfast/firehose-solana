package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/streamingfast/bstream/transform"
	"github.com/streamingfast/dstore"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream"
	dauthAuthenticator "github.com/streamingfast/dauth/authenticator"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/dmetering"
	"github.com/streamingfast/dmetrics"
	firehoseApp "github.com/streamingfast/firehose/app/firehose"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/substreams/client"
	pbsubstreams "github.com/streamingfast/substreams/pb/sf/substreams/v1"
	substreamsService "github.com/streamingfast/substreams/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
			cmd.Flags().StringSlice("firehose-blocks-store-urls", nil, "If non-empty, overrides common-merged-blocks-store-url with a list of blocks stores")
			cmd.Flags().Duration("firehose-real-time-tolerance", 1*time.Minute, "Firehose will became alive if now - block time is smaller then tolerance")

			cmd.Flags().Bool("substreams-enabled", false, "Whether to enable substreams")
			cmd.Flags().Bool("substreams-partial-mode-enabled", false, "Whether to enable partial stores generation support on this instance (usually for internal deployments only)")
			cmd.Flags().String("substreams-state-store-url", "{sf-data-dir}/localdata", "where substreams state data are stored")
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
			sfDataDir := runtime.AbsDataDir
			tracker := runtime.Tracker.Clone()
			blockstreamAddr := viper.GetString("common-live-blocks-addr")
			if blockstreamAddr != "" {
				tracker.AddGetter(bstream.BlockStreamLIBTarget, bstream.StreamLIBBlockRefGetter(blockstreamAddr))
			}

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

			firehoseBlocksStoreURLs := viper.GetStringSlice("firehose-blocks-store-urls")
			if len(firehoseBlocksStoreURLs) == 0 {
				firehoseBlocksStoreURLs = []string{viper.GetString("common-merged-blocks-store-url")}
			} else if len(firehoseBlocksStoreURLs) == 1 && strings.Contains(firehoseBlocksStoreURLs[0], ",") {
				// Providing multiple elements from config doesn't work with `viper.GetStringSlice`, so let's also handle the case where a single element has separator
				firehoseBlocksStoreURLs = strings.Split(firehoseBlocksStoreURLs[0], ",")
			}

			for _, url := range firehoseBlocksStoreURLs {
				url = MustReplaceDataDir(sfDataDir, url)
			}

			shutdownSignalDelay := viper.GetDuration("common-system-shutdown-signal-delay")
			grcpShutdownGracePeriod := time.Duration(0)
			if shutdownSignalDelay.Seconds() > 5 {
				grcpShutdownGracePeriod = shutdownSignalDelay - (5 * time.Second)
			}

			var registerServiceExt firehoseApp.RegisterServiceExtensionFunc
			if viper.GetBool("substreams-enabled") {

				stateStore, err := dstore.NewStore(MustReplaceDataDir(sfDataDir, viper.GetString("substreams-state-store-url")), "", "", true)
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

				ssClientFactory := func() (pbsubstreams.StreamClient, []grpc.CallOption, error) {
					endpoint := viper.GetString("substreams-client-endpoint")
					if endpoint == "" {
						endpoint = viper.GetString("firehose-grpc-listen-addr")
					}

					return client.NewSubstreamsClient(
						endpoint,
						os.ExpandEnv(viper.GetString("substreams-client-jwt")),
						ssClientInsecure,
						ssClientPlaintext,
					)
				}

				sss := substreamsService.New(
					stateStore,
					"sf.solana.type.v1.Block",
					ssClientFactory,
					viper.GetInt("substreams-sub-request-parallel-jobs"),
					viper.GetInt("substreams-sub-request-block-range-size"),
					opts...,
				)

				registerServiceExt = sss.Register
			}

			registry := transform.NewRegistry()

			return firehoseApp.New(appLogger, &firehoseApp.Config{
				BlockStoreURLs:          firehoseBlocksStoreURLs,
				BlockStreamAddr:         blockstreamAddr,
				GRPCListenAddr:          viper.GetString("firehose-grpc-listen-addr"),
				GRPCShutdownGracePeriod: grcpShutdownGracePeriod,
				RealtimeTolerance:       viper.GetDuration("firehose-real-time-tolerance"),
				//IrreversibleBlocksIndexStoreURL: viper.GetString("firehose-irreversible-blocks-index-url"),
				//IrreversibleBlocksBundleSizes:   bundleSizes,
			}, &firehoseApp.Modules{
				Authenticator:            authenticator,
				HeadTimeDriftMetric:      headTimeDriftmetric,
				HeadBlockNumberMetric:    headBlockNumMetric,
				Tracker:                  tracker,
				TransformRegistry:        registry,
				RegisterServiceExtension: registerServiceExt,
			}), nil
		},
	})
}
