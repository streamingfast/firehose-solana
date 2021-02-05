package cli

import (
	"github.com/dfuse-io/dfuse-solana/snapshotter/app/snapshotter"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "snapshotter",
		Title:       "snapshotter",
		Description: "Manage solana snapshot",
		MetricsID:   "snapshotter",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/snapshotter.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("snapshotter-solana-snapshot-bucket", "gs://mainnet-beta-ledger-us-west1", "bucket where solana snapshot are stored")
			cmd.Flags().String("snapshotter-solana-snapshot-prefix", "", "mainnet-beta-ledger-us-west1")
			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) (err error) {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			return snapshotter.New(
				&snapshotter.Config{
					Bucket: viper.GetString("snapshotter-solana-snapshot-bucket"),
					Prefix: viper.GetString("snapshotter-solana-snapshot-prefix"),
				},
			), nil
		},
	})
}
