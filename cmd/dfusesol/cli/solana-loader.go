package cli

import (
	solanadbLoaderApp "github.com/dfuse-io/dfuse-solana/solanadb-loader/app/solanadb-loader"

	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "solana-loader",
		Title:       "Solana loader",
		Description: "Solana's main database",
		MetricsID:   "solana-loader",
		Logger: launcher.NewLoggingDef(
			"github.com/dfuse-io/dfuse-solana/(solanadb|solanadb-loader).*",
			[]zapcore.Level{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel},
		),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("solanadb-loader-dsn", "live", "kvdb connection string to solana databse database")
			cmd.Flags().Uint64("solanadb-loader-batch-size", 100, "Max number of slots batched together for database write. Slots are not batched when close (<25sec) to head.")
			cmd.Flags().Uint64("solanadb-loader-start-block-num", 0, "Block number where we start processing")
			cmd.Flags().String("solanadb-loader-http-listen-addr", SolanaLoaderHTTPServingAddr, "Listen address for /healthz endpoint")
			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) error {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir

			return solanadbLoaderApp.New(&solanadbLoaderApp.Config{
				BlockStreamAddr: mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blockstream-addr")),
				BatchSize:       viper.GetUint64("solanadb-loader-batch-size"),
				StartBlock:      viper.GetUint64("solanadb-loader-start-block-num"),
				KvdbDsn:         mustReplaceDataDir(dfuseDataDir, viper.GetString("solanadb-loader-dsn")),
				HTTPListenAddr:  viper.GetString("solanadb-loader-http-listen-addr"),
			}), nil
		},
	})
}
