package codec

import "github.com/streamingfast/logging"

var zlog, _ = logging.PackageLogger("firesol", "github.com/streamingfast/firehose-solana/codec")

func init() {
	logging.InstantiateLoggers()
}

type ObjectReader func() (interface{}, error)
