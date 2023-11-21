package rpc

import (
	"github.com/spf13/cobra"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewBigTableCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{Use: "rpc", Short: "rpc"}

	cmd.AddCommand(newPrintCmd(logger, tracer))
	return cmd
}
