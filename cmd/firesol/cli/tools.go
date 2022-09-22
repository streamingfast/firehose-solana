package cli

import "github.com/streamingfast/firehose-solana/tools"

func init() {
	RootCmd.AddCommand(tools.Cmd)
}
