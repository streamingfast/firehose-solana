package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/block/fetcher"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

func NewPrintBlockCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "print-block <slot>",
		Short: "fetch and print a single block from rpc endpoint as JSON",
		Args:  cobra.ExactArgs(1),
		RunE:  printBlockRunE(logger, tracer),
	}

	cmd.Flags().StringArray("endpoints", []string{}, "List of endpoints to use to fetch different method calls")
	cmd.Flags().String("network", "mainnet", "network to fetch from (mainnet, devnet, testnet) -- only used to patch a known issue on some slots")

	return cmd
}

func printBlockRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		requestedSlot, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse slot %s: %w", args[0], err)
		}

		rpcEndpoints := sflags.MustGetStringArray(cmd, "endpoints")
		if len(rpcEndpoints) == 0 {
			return fmt.Errorf("at least one RPC endpoint is required (use --endpoints flag)")
		}

		// Create RPC client (just use the first endpoint for simplicity)
		client := rpc.New(rpcEndpoints[0])

		// Determine network
		var isMainnet bool
		switch sflags.MustGetString(cmd, "network") {
		case "mainnet", "mainnet-beta":
			isMainnet = true
		}

		// Create fetcher without saveToFile
		blockFetcher := fetcher.NewRPC(time.Second, isMainnet, false, logger)

		// Fetch the block using the same method as in rpc.go:111-112
		logger.Info("fetching single block", zap.Uint64("slot", requestedSlot))
		blockResult, skip, err := blockFetcher.Fetch(ctx, client, requestedSlot)
		if err != nil {
			return fmt.Errorf("fetching block: %w", err)
		}

		if skip {
			logger.Info("block was skipped", zap.Uint64("slot", requestedSlot))
			return nil
		}

		// Extract pbsol.Block from the pbbstream.Block payload
		solBlock := &pbsol.Block{}
		if err := blockResult.Payload.UnmarshalTo(solBlock); err != nil {
			return fmt.Errorf("unmarshaling payload to pbsol.Block: %w", err)
		}

		// Marshal the pbsol.Block to JSON using protojson for better formatting
		marshaler := protojson.MarshalOptions{
			Indent:          "  ",
			EmitUnpopulated: false,
		}
		jsonData, err := marshaler.Marshal(solBlock)
		if err != nil {
			return fmt.Errorf("marshaling block to JSON: %w", err)
		}
		fmt.Println(string(jsonData))

		return nil
	}
}
