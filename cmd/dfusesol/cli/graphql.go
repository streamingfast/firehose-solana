package cli

import (
	"github.com/dfuse-io/dfuse-solana/graphql/app/graphql"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/dfuse-io/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	appLogger := zap.NewNop()

	logging.Register("github.com/dfuse-io/dfuse-solana/graphql", &appLogger)

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
			cmd.Flags().String("graphql-config-name", "mainnet", "")
			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			return graphql.New(&graphql.Config{
				Name: viper.GetString("graphql-config-name"),
			}, &graphql.Modules{}), nil
		},
	})
}
