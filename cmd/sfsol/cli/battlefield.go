package cli

import "github.com/streamingfast/firehose-solana/battlefield"

func init() {
	RootCmd.AddCommand(battlefield.Cmd)
}
