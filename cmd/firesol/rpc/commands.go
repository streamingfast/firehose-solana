package rpc

import (
	"github.com/spf13/cobra"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func NewBigTableCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{Use: "poller", Short: "poller"}

	cmd.AddCommand(NewPollerCmd(logger, tracer))
	return cmd
}
