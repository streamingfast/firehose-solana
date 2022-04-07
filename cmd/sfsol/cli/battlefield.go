package cli

import "github.com/streamingfast/sf-solana/battlefield"

func init() {
	RootCmd.AddCommand(battlefield.Cmd)
}
