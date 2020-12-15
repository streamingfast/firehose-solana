package cli

import (
	"github.com/dfuse-io/dfuse-solana/graphql/app/graphql"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "graphql",
		Title:       "graphql",
		Description: "graphql",
		MetricsID:   "graphql",
		Logger: launcher.NewLoggingDef(
			"github.com/dfuse-io/dfuse-solana/graphql.*",
			[]zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel},
		),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("graphql-rpc-url", "http://api.mainnet-beta.solana.com:80/rpc", "")
			cmd.Flags().String("graphql-rpc-ws-url", "ws://api.mainnet-beta.solana.com:80/rpc", "")
			cmd.Flags().String("graphql-http-listen-addr", ":8080", "")
			cmd.Flags().String("graphql-config-name", "mainnet", "")
			cmd.Flags().Uint64("graphql-slot-offset", 100, "number of slots offset")
			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			return graphql.New(&graphql.Config{
				RPCURL:            viper.GetString("graphql-rpc-url"),
				RPCWSURL:          viper.GetString("graphql-rpc-ws-url"),
				HTTPListenAddress: viper.GetString("graphql-http-listen-addr"),
				SlotOffset:        viper.GetUint64("graphql-slot-offset"),
			}, &graphql.Modules{}), nil
		},
	})
}
