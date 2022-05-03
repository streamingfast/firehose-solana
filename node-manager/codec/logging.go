package codec

import (
	"github.com/mr-tron/base58"
	"go.uber.org/zap"
)

type zapBase58 []byte

func (b zapBase58) String() string {
	return base58.Encode([]byte(b))
}

func ZapBase58(key string, input []byte) zap.Field {
	return zap.Stringer(key, zapBase58(input))
}
