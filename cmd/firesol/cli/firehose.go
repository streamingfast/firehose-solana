package cli

import (
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/bstream/transform"
	"github.com/streamingfast/dauth"
	discoveryservice "github.com/streamingfast/dgrpc/server/discovery-service"
	"github.com/streamingfast/dlauncher/launcher"
	"github.com/streamingfast/dmetering"
	"github.com/streamingfast/dmetrics"
	firehoseApp "github.com/streamingfast/firehose/app/firehose"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var metricset = dmetrics.NewSet()
var headBlockNumMetric = metricset.NewHeadBlockNumber("firehose")
var headTimeDriftMetric = metricset.NewHeadTimeDrift("firehose")

func init() {
	appLogger := zap.NewNop()
	appLogger, _ = logging.PackageLogger("firehose", "github.com/streamingfast/firehose-solana/firehose")

	launcher.RegisterApp(zlog, &launcher.AppDef{
		ID:          "firehose",
		Title:       "Block Firehose",
		Description: "Provides on-demand filtered blocks, depends on common-merged-blocks-store-url and common-live-blocks-addr",
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("firehose-grpc-listen-addr", FirehoseGRPCServingAddr, "Address on which the Firehose will listen")
			cmd.Flags().String("firehose-discovery-service-url", "", "url to configure the grpc discovery service") //traffic-director://xds?vpc_network=vpc-global&use_xds_reds=true

			return nil
		},

		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			blockstreamAddr := viper.GetString("common-live-blocks-addr")

			// FIXME: That should be a shared dependencies across `EOSIO on StreamingFast`
			authenticator, err := dauth.New(viper.GetString("common-auth-plugin"))
			if err != nil {
				return nil, fmt.Errorf("unable to initialize dauth: %w", err)
			}

			// FIXME: That should be a shared dependencies across `EOSIO on StreamingFast` it will avoid the need to call `dmetering.SetDefaultMeter`
			metering, err := dmetering.New(viper.GetString("common-metering-plugin"), appLogger)
			if err != nil {
				return nil, fmt.Errorf("unable to initialize dmetering: %w", err)
			}
			dmetering.SetDefaultEmitter(metering)

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

			rawServiceDiscoveryURL := viper.GetString("firehose-discovery-service-url")
			var serviceDiscoveryURL *url.URL
			if rawServiceDiscoveryURL != "" {
				serviceDiscoveryURL, err = url.Parse(rawServiceDiscoveryURL)
				if err != nil {
					return nil, fmt.Errorf("unable to parse discovery service url: %w", err)
				}
				err = discoveryservice.Bootstrap(serviceDiscoveryURL)
				if err != nil {
					return nil, fmt.Errorf("unable to bootstrap discovery service: %w", err)
				}
			}

			registry := transform.NewRegistry()

			return firehoseApp.New(appLogger, &firehoseApp.Config{
				MergedBlocksStoreURL:    mergedBlocksStoreURL,
				OneBlocksStoreURL:       oneBlockStoreURL,
				ForkedBlocksStoreURL:    forkedBlocksStoreURL,
				BlockStreamAddr:         blockstreamAddr,
				GRPCListenAddr:          viper.GetString("firehose-grpc-listen-addr"),
				ServiceDiscoveryURL:     serviceDiscoveryURL,
				GRPCShutdownGracePeriod: grcpShutdownGracePeriod,
			}, &firehoseApp.Modules{
				Authenticator:            authenticator,
				HeadTimeDriftMetric:      headTimeDriftMetric,
				HeadBlockNumberMetric:    headBlockNumMetric,
				TransformRegistry:        registry,
				RegisterServiceExtension: registerServiceExt,
			}), nil
		},
	})
}
