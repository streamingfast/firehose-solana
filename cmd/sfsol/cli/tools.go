package cli

import "github.com/streamingfast/sf-solana/tools"

func init() {
	RootCmd.AddCommand(tools.Cmd)
}
