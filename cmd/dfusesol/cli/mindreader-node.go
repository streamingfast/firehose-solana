package cli

import (
	"github.com/dfuse-io/dfuse-solana/mindreader/app/mindreader"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/dfuse-io/logging"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	appLogger := zap.NewNop()
	gethLogger := zap.NewNop()

	logging.Register("github.com/dfuse-io/dfuse-ethereum/mindreader-node", &appLogger)
	logging.Register("github.com/dfuse-io/dfuse-ethereum/mindreader-node/geth", &gethLogger)

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "mindreader-node",
		Title:       "Mindreader Node",
		Description: "Mindreader node ",
		MetricsID:   "mindreader-node",
		Logger: launcher.NewLoggingDef(
			"github.com/dfuse-io/dfuse-solana/mindreader-node.*",
			[]zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel},
		),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Bool("mindreader-node-log-to-zap", true, "Enable all node logs to transit into app's logger directly, when false, prints node logs directly to stdout")

			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			return mindreader.New(&mindreader.Config{}, &mindreader.Modules{}), nil
		},
	})
}
