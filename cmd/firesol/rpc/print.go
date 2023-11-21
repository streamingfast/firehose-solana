package rpc

import (
	"github.com/spf13/cobra"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func newPrintCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "print <block_num>",
		Short: "print a block from rpc request",
		Args:  cobra.ExactArgs(1),
	}
	cmd.AddCommand(newPrintBlocksCmd(logger, tracer))
	return cmd
}

func newPrintBlocksCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "block",
		RunE:  printBlockRunE(logger, tracer),
	}
	return cmd
}

func printBlockRunE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) error {
		//ctx := cmd.Context()
		//
		//blockNumStr := args[0]
		//btProject := sflags.MustGetString(cmd, "bt-project")
		//btInstance := sflags.MustGetString(cmd, "bt-instance")
		//
		//logger.Info("retrieving from bigtable",
		//	zap.String("block_num", blockNumStr),
		//	zap.String("bt_project", btProject),
		//	zap.String("bt_instance", btInstance),
		//)
		//
		return nil
	}
}
