package cli

import "github.com/dfuse-io/dfuse-solana/tools"

func init() {
	RootCmd.AddCommand(tools.Cmd)
}
