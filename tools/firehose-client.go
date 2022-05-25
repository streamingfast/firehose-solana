package tools

import (
	"github.com/spf13/cobra"
	//"github.com/streamingfast/eth-go"
	//pbtransform "github.com/streamingfast/sf-ethereum/types/pb/sf/ethereum/transform/v1"
	sftools "github.com/streamingfast/sf-tools"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	firehoseClientCmd := sftools.GetFirehoseClientCmd(zlog, tracer, transformsSetter)
	Cmd.AddCommand(firehoseClientCmd)
}

// no transforms on arweave yet
var transformsSetter = func(cmd *cobra.Command) (transforms []*anypb.Any, err error) {
	return nil, nil
}
