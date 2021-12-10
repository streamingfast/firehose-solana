package transform

import (
	"github.com/streamingfast/bstream/transform"
	pbtransforms "github.com/streamingfast/sf-solana/pb/sf/solana/transforms/v1"
	"google.golang.org/protobuf/proto"
)

func init() {
	transform.Register(proto.MessageName(&pbtransforms.ProgramFilter{}), NewProgramFilterFactory)
}
