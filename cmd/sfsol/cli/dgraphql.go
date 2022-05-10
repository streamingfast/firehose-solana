package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	dgraphqlApp "github.com/streamingfast/dgraphql/app/dgraphql"
	"github.com/streamingfast/dlauncher/launcher"
	dgraphqlSol "github.com/streamingfast/sf-solana/dgraphql"
)

func init() {
	launcher.RegisterApp(zlog, &launcher.AppDef{
		ID:          "dgraphql",
		Title:       "GraphQL",
		Description: "Serves GraphQL queries to clients",
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("dgraphql-http-addr", DgraphqlHTTPServingAddr, "TCP Listener addr for http")
			cmd.Flags().String("dgraphql-grpc-addr", DgraphqlGRPCServingAddr, "TCP Listener addr for gRPC")
			cmd.Flags().Bool("dgraphql-disable-authentication", true, "Disable authentication for both grpc and http services")
			cmd.Flags().Bool("dgraphql-override-trace-id", false, "Flag to override trace id or not")
			cmd.Flags().String("dgraphql-auth-url", "null://", "Auth URL used to configure the StreamingFast JavaScript client")
			cmd.Flags().String("dgraphql-api-key", "web_0000", "API key used in GraphiQL")
			cmd.Flags().String("dgraphql-serumhist-grpc-addr", SerumHistoryGRPCServingAddr, "Address where to reach the Serum History gRPC service")
			cmd.Flags().Uint64("dgraphql-slot-offset", 100, "Number of slots offset")
			cmd.Flags().String("dgraphql-tokens-file-url", "gs://staging.dfuseio-global.appspot.com/sol-tokens/sol-mainnet-v1.jsonl", "JSONL file containing list of known tokens")
			cmd.Flags().String("dgraphql-markets-file-url", "gs://staging.dfuseio-global.appspot.com/sol-markets/sol-mainnet-v1.jsonl", "JSONL file containing list of known markets")
			cmd.Flags().String("dgraphql-serumviz-bigquery-dsn", "bigquery://dfuse-development-tools/us/serum", "The BigQuery DSN to use to retrieve data for Serum Vizualisation needs via BigQuery.")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			return dgraphqlSol.NewApp(&dgraphqlSol.Config{
				// Solana specifc configs
				RatelimiterPlugin:     viper.GetString("common-ratelimiter-plugin"),
				RPCEndpointAddr:       viper.GetString("common-rpc-endpoint"),
				RPCWSEndpointAddr:     viper.GetString("common-rpc-ws-endpoint"),
				SlotOffset:            viper.GetUint64("dgraphql-slot-offset"),
				SerumHistoryAddr:      viper.GetString("dgraphql-serumhist-grpc-addr"),
				TokensFileURL:         viper.GetString("dgraphql-tokens-file-url"),
				MarketFileURL:         viper.GetString("dgraphql-markets-file-url"),
				SerumhistAnalyticsDSN: viper.GetString("dgraphql-serumviz-bigquery-dsn"),
				Config: dgraphqlApp.Config{
					// Base dgraphql configs
					Protocol:        "sol",
					HTTPListenAddr:  viper.GetString("dgraphql-http-addr"),
					GRPCListenAddr:  viper.GetString("dgraphql-grpc-addr"),
					AuthPlugin:      viper.GetString("common-auth-plugin"),
					MeteringPlugin:  viper.GetString("common-metering-plugin"),
					NetworkID:       viper.GetString("common-sf-network-id"),
					OverrideTraceID: viper.GetBool("dgraphql-override-trace-id"),
					JwtIssuerURL:    viper.GetString("dgraphql-auth-url"),
					APIKey:          viper.GetString("dgraphql-api-key"),
				},
			})
		},
	})
}
