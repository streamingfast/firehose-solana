package tools

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{Use: "tools", Short: "Developer tools related to firesol"}

func mustGetBool(cmd *cobra.Command, flagName string) bool {
	val, err := cmd.Flags().GetBool(flagName)
	if err != nil {
		panic(fmt.Sprintf("flags: couldn't find flag %q", flagName))
	}
	return val
}
