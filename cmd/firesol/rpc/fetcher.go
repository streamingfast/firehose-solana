package rpc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/pflag"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-core/blockpoller"
	firecoreRPC "github.com/streamingfast/firehose-core/rpc"
	"github.com/streamingfast/firehose-solana/block/fetcher"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewFetchCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rpc <first-streamable-block>",
		Short: "fetch blocks from rpc endpoint",
		Args:  cobra.ExactArgs(1),
		RunE:  fetchRunE(logger, tracer),
	}

	cmd.Flags().StringArray("endpoints", []string{}, "List of endpoints to use to fetch different method calls")
	cmd.Flags().String("state-dir", "/data/poller", "interval between fetch")
	cmd.Flags().Duration("interval-between-fetch", 0, "interval between fetch")
	cmd.Flags().Duration("latest-block-retry-interval", time.Second, "interval between fetch")
	cmd.Flags().Duration("max-block-fetch-duration", 3*time.Second, "maximum delay before considering a block fetch as failed")
	cmd.Flags().Duration("interval-between-clients-sort", 10*time.Minute, "interval between sorting clients base on their head block")
	cmd.Flags().Int("block-fetch-batch-size", 10, "Number of blocks to fetch in a single batch")
	cmd.Flags().String("network", "mainnet", "network to fetch from (mainnet, devnet, testnet) -- only used to patch a known issue on some slots")

	return cmd
}

func fetchRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) (err error) {
		stateDir := sflags.MustGetString(cmd, "state-dir")

		startBlock, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse first streamable block %d: %w", startBlock, err)
		}

		fetchInterval := sflags.MustGetDuration(cmd, "interval-between-fetch")
		maxBlockFetchDuration := sflags.MustGetDuration(cmd, "max-block-fetch-duration")
		intervalBetweenClientsSort := sflags.MustGetDuration(cmd, "interval-between-clients-sort")

		logger.Info("launching firehose-solana poller")
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			logger.Info("flag", zap.String("name", flag.Name), zap.String("value", flag.Value.String()))
		})

		rpcEndpoints := sflags.MustGetStringArray(cmd, "endpoints")
		rpcClients := firecoreRPC.NewClients[*rpc.Client](maxBlockFetchDuration, firecoreRPC.NewStickyRollingStrategy[*rpc.Client](), logger)
		for _, rpcEndpoint := range rpcEndpoints {
			client := rpc.New(rpcEndpoint)
			rpcClients.Add(client)
		}

		latestBlockRetryInterval := sflags.MustGetDuration(cmd, "latest-block-retry-interval")
		var isMainnet bool
		switch sflags.MustGetString(cmd, "network") {
		case "mainnet", "mainnet-beta":
			isMainnet = true
		}

		blockFetcher := fetcher.NewRPC(latestBlockRetryInterval, isMainnet, logger)
		rpcClients.StartSorting(cmd.Context(), firecoreRPC.SortDirectionDescending, blockFetcher, intervalBetweenClientsSort)

		poller := blockpoller.New[*rpc.Client](
			blockFetcher,
			blockpoller.NewFireBlockHandler("type.googleapis.com/sf.solana.type.v1.Block"),
			rpcClients,
			blockpoller.WithLogger[*rpc.Client](logger),
			blockpoller.WithStoringState[*rpc.Client](stateDir),
			blockpoller.WithDelayBetweenFetch[*rpc.Client](fetchInterval),
		)

		err = poller.Run(startBlock, nil, sflags.MustGetInt(cmd, "block-fetch-batch-size"))
		if err != nil {
			return fmt.Errorf("running poller: %w", err)
		}

		return nil
	}
}
