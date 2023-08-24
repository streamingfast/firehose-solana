package codec

import (
	firehose_solana "github.com/streamingfast/firehose-solana"
	"github.com/streamingfast/logging"
)

var zlog, _ = logging.PackageLogger("firesol", "github.com/streamingfast/firehose-solana/codec.test")

type ObjectReader func() (interface{}, error)

func init() {
	logging.InstantiateLoggers()
	firehose_solana.TestingInitBstream()
}
