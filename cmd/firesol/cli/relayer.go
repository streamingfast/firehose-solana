package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dlauncher/launcher"
	relayerApp "github.com/streamingfast/relayer/app/relayer"
)

func init() {
	// Relayer
	launcher.RegisterApp(zlog, &launcher.AppDef{
		ID:          "relayer",
		Title:       "Relayer",
		Description: "Serves blocks as a stream, with a buffer",
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("relayer-grpc-listen-addr", RelayerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().StringSlice("relayer-source", []string{ReaderNodeGRPCAddr}, "List of Blockstream sources (readers) to connect to for live block feeds (repeat flag as needed)")
			cmd.Flags().Duration("relayer-max-source-latency", 10*time.Minute, "Max latency tolerated to connect to a source")

			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			_, oneBlocksStoreURL, _, err := getCommonStoresURLs(runtime.AbsDataDir)
			if err != nil {
				return nil, fmt.Errorf("failed to get common block store URL: %w", err)
			}

			return relayerApp.New(&relayerApp.Config{
				SourcesAddr:      viper.GetStringSlice("relayer-source"),
				OneBlocksURL:     oneBlocksStoreURL,
				GRPCListenAddr:   viper.GetString("relayer-grpc-listen-addr"),
				MaxSourceLatency: viper.GetDuration("relayer-max-source-latency"),
			}), nil
		},
	})
}
