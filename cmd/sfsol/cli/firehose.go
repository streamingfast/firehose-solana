package cli

import (
	"fmt"
	"strings"
	"time"

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
	substreamsService "github.com/streamingfast/substreams/service"
	"go.uber.org/zap"
)

var metricset = dmetrics.NewSet()
var headBlockNumMetric = metricset.NewHeadBlockNumber("firehose")
var headTimeDriftmetric = metricset.NewHeadTimeDrift("firehose")

func init() {
	appLogger := zap.NewNop()
	appLogger, _ = logging.PackageLogger("firehose", "github.com/streamingfast/sf-solana/firehose")

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "firehose",
		Title:       "Block Firehose",
		Description: "Provides on-demand filtered blocks, depends on common-blocks-store-url and common-blockstream-addr",
		MetricsID:   "merged-filter",
		Logger:      launcher.NewLoggingDef("github.com/streamingfast/sf-solana/firehose.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("firehose-grpc-listen-addr", FirehoseGRPCServingAddr, "Address on which the firehose will listen")
			cmd.Flags().StringSlice("firehose-blocks-store-urls", nil, "If non-empty, overrides common-blocks-store-url with a list of blocks stores")
			cmd.Flags().Duration("firehose-real-time-tolerance", 1*time.Minute, "firehose will became alive if now - block time is smaller then tolerance")
			cmd.Flags().Bool("substreams-enabled", false, "Whether to enable substreams")
			cmd.Flags().Bool("substreams-partial-mode-enabled", false, "Whether to enable partial stores generation support on this instance (usually for internal deployments only)")
			cmd.Flags().String("substreams-state-store-url", "./localdata", "where substreams state data are stored")
			cmd.Flags().Uint64("substreams-stores-save-interval", uint64(10000), "Interval in blocks at which to save store snapshots")
			return nil
		},

		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			sfDataDir := runtime.AbsDataDir
			tracker := runtime.Tracker.Clone()
			blockstreamAddr := viper.GetString("common-blockstream-addr")
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
				firehoseBlocksStoreURLs = []string{viper.GetString("common-blocks-store-url")}
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
				stateStore, err := dstore.NewStore(viper.GetString("substreams-state-store-url"), "", "", false)
				if err != nil {
					return nil, fmt.Errorf("setting up state store for data: %w", err)
				}

				opts := []substreamsService.Option{
					substreamsService.WithStoresSaveInterval(viper.GetUint64("substreams-stores-save-interval")),
				}

				if viper.GetBool("substreams-partial-mode-enabled") {
					opts = append(opts, substreamsService.WithPartialMode())
				}
				sss := substreamsService.New(
					stateStore,
					"sf.solana.type.v1.Block",
					opts...,
				)

				registerServiceExt = sss.Register
			}

			return firehoseApp.New(appLogger, &firehoseApp.Config{
				BlockStoreURLs:          firehoseBlocksStoreURLs,
				BlockStreamAddr:         blockstreamAddr,
				GRPCListenAddr:          viper.GetString("firehose-grpc-listen-addr"),
				GRPCShutdownGracePeriod: grcpShutdownGracePeriod,
				RealtimeTolerance:       viper.GetDuration("firehose-real-time-tolerance"),
			}, &firehoseApp.Modules{
				Authenticator:            authenticator,
				HeadTimeDriftMetric:      headTimeDriftmetric,
				HeadBlockNumberMetric:    headBlockNumMetric,
				Tracker:                  tracker,
				RegisterServiceExtension: registerServiceExt,
			}), nil
		},
	})
}
