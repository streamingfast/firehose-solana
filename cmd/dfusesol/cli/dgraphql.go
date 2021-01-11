package cli

import (
	dgraphqlSol "github.com/dfuse-io/dfuse-solana/dgraphql"
	dgraphqlApp "github.com/dfuse-io/dgraphql/app/dgraphql"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "dgraphql",
		Title:       "GraphQL",
		Description: "Serves GraphQL queries to clients",
		MetricsID:   "dgraphql",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/(dgraphql.*|dfuse-ethereum/dgraphql.*)", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("dgraphql-http-addr", DgraphqlHTTPServingAddr, "TCP Listener addr for http")
			cmd.Flags().String("dgraphql-grpc-addr", DgraphqlGRPCServingAddr, "TCP Listener addr for gRPC")
			cmd.Flags().Bool("dgraphql-disable-authentication", true, "Disable authentication for both grpc and http services")
			cmd.Flags().Bool("dgraphql-override-trace-id", false, "Flag to override trace id or not")
			cmd.Flags().String("dgraphql-auth-url", "null://", "Auth URL used to configure the dfuse js client")
			cmd.Flags().String("dgraphql-api-key", "web_0000", "API key used in GraphiQL")
			cmd.Flags().String("dgraphql-serum-hist-addr", SerumHistoryGRPCServingAddr, "Address where to reach the Serum History gRPC service")
			cmd.Flags().Uint64("dgraphql-slot-offset", 100, "Number of slots offset")
			cmd.Flags().String("dgraphql-api-key", "web_0000", "API key used in GraphiQL")
			cmd.Flags().String("dgraphql-token-list-url", "gs://staging.dfuseio-global.appspot.com/sol-tokens/sol-mainnet-v1.jsonl", "JSONL file containing list of known tokens")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			return dgraphqlSol.NewApp(&dgraphqlSol.Config{
				// Solana specifc configs
				RatelimiterPlugin: viper.GetString("common-ratelimiter-plugin"),
				RPCEndpointAddr:   viper.GetString("common-rpc-endpoint"),
				RPCWSEndpointAddr: viper.GetString("common-rpc-ws-endpoint"),
				SlotOffset:        viper.GetUint64("dgraphql-slot-offset"),
				SerumHistoryAddr:  viper.GetString("dgraphql-serum-hist-addr"),
				TokenListURL:      viper.GetString("dgraphql-token-list-url"),
				Config: dgraphqlApp.Config{
					// Base dgraphql configs
					Protocol:        "sol",
					HTTPListenAddr:  viper.GetString("dgraphql-http-addr"),
					GRPCListenAddr:  viper.GetString("dgraphql-grpc-addr"),
					AuthPlugin:      viper.GetString("common-auth-plugin"),
					MeteringPlugin:  viper.GetString("common-metering-plugin"),
					NetworkID:       viper.GetString("common-dfuse-network-id"),
					OverrideTraceID: viper.GetBool("dgraphql-override-trace-id"),
					JwtIssuerURL:    viper.GetString("dgraphql-auth-url"),
					APIKey:          viper.GetString("dgraphql-api-key"),
				},
			})
		},
	})
}
