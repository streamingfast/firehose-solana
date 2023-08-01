package main

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newToolsBatchFileCmd(logger *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{Use: "batch-files", Short: "batch Files related commands"}
	cmd.AddCommand(newToolsBatchFileReadCmd(logger))
	return cmd
}
