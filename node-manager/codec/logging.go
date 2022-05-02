package codec

import (
	"github.com/mr-tron/base58"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog, tracer = logging.PackageLogger("codec", "github.com/streamingfast/sf-solana/node-manager/codec")

type zapBase58 []byte

func (b zapBase58) String() string {
	return base58.Encode([]byte(b))
}

func ZapBase58(key string, input []byte) zap.Field {
	return zap.Stringer(key, zapBase58(input))
}
