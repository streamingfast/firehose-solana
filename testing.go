package firehose_solana

import (
	"github.com/streamingfast/bstream"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"google.golang.org/protobuf/proto"
)

func TestingInitBstream() {
	// Should be aligned with firecore.Chain as defined in `cmd/firesol/main.go``
	bstream.InitGeneric("NEA", 1, func() proto.Message {
		return new(pbsol.Block)
	})
}
