package cli

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dlauncher/launcher"
	mergerApp "github.com/streamingfast/merger/app/merger"
)

func init() {
	launcher.RegisterApp(zlog, &launcher.AppDef{
		ID:          "merger",
		Title:       "Merger",
		Description: "Produces merged block files from single-block files",
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Duration("merger-time-between-store-lookups", 5*time.Second, "delay between source store polling (should be higher for remote storage)")
			cmd.Flags().Duration("merger-time-between-store-pruning", time.Minute, "Delay between source store pruning loops")
			cmd.Flags().Uint64("merger-prune-forked-blocks-after", 50000, "Number of blocks that must pass before we delete old forks (one-block-files lingering)")
			cmd.Flags().String("merger-grpc-listen-addr", MergerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Uint64("merger-stop-block", 0, "if non-zero, merger will trigger shutdown when blocks have been merged up to this block")
			return nil
		},
		InitFunc: func(runtime *launcher.Runtime) (err error) {
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			mergedBlocksStoreURL, oneBlockStoreURL, forkedBlocksStoreURL, err := getCommonStoresURLs(runtime.AbsDataDir)
			if err != nil {
				return nil, err
			}

			return mergerApp.New(&mergerApp.Config{
				StorageMergedBlocksFilesPath: mergedBlocksStoreURL,
				StorageOneBlockFilesPath:     oneBlockStoreURL,
				StorageForkedBlocksFilesPath: forkedBlocksStoreURL,
				GRPCListenAddr:               viper.GetString("merger-grpc-listen-addr"),
				PruneForkedBlocksAfter:       viper.GetUint64("merger-prune-forked-blocks-after"),
				StopBlock:                    viper.GetUint64("merger-stop-block"),
				TimeBetweenPruning:           viper.GetDuration("merger-time-between-store-pruning"),
				TimeBetweenPolling:           viper.GetDuration("merger-time-between-store-lookups"),
			}), nil
		},
	})
}
