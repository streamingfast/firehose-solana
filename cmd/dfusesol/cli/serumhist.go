package cli

import (
	serumhistApp "github.com/dfuse-io/dfuse-solana/serumhist/app/serumhist"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// Accounthist
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "serumhist",
		Title:       "Serum History Server",
		Description: "Serves fills for pubkey and or market",
		MetricsID:   "serumhist",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-solana/serumhist/.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("serumhist-dsn", SerumHistDSN, "kvdb connection string to solana databse database")
			cmd.Flags().Uint64("serumhist-start-block-num", 0, "Block number where we start processing")
			cmd.Flags().String("serumhist-grpc-listen-addr", SerumHistoryGRPCServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("serumhist-http-listen-addr", SerumHistoryHTTPServingAddr, "Listen address for /healthz endpoint")
			cmd.Flags().Bool("serumhist-enable-injector-mode", true, "Enable mode where blocks are ingested, processed and saved to the database, when false, no write operations happen.")
			cmd.Flags().Bool("serumhist-enable-server-mode", true, "Enable mode where the gRPC server is started and answers request(s), when false, the server is disabled and no request(s) will be handled.")
			cmd.Flags().Uint64("serumhist-flush-slots-interval", 100, "Flush to storage each X blocks.  Use 1 when live. Use a high number in batch, serves as checkpointing between restarts.")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			return serumhistApp.New(&serumhistApp.Config{
				BlockStreamAddr:   viper.GetString("common-blockstream-addr"),
				FLushSlotInterval: viper.GetUint64("serumhist-flush-slots-interval"),
				StartBlock:        viper.GetUint64("serumhist-start-block-num"),
				KvdbDsn:           mustReplaceDataDir(dfuseDataDir, viper.GetString("serumhist-dsn")),
				EnableInjector:    viper.GetBool("serumhist-enable-injector-mode"),
				EnableServer:      viper.GetBool("serumhist-enable-server-mode"),
				GRPCListenAddr:    viper.GetString("serumhist-grpc-listen-addr"),
				HTTPListenAddr:    viper.GetString("serumhist-http-listen-addr"),
			}), nil
		},
	})
}
