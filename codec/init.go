package codec

import (
	"github.com/streamingfast/bstream"
)

func init() {
	bstream.GetBlockWriterFactory = bstream.BlockWriterFactoryFunc(BlockWriterFactory)
	bstream.GetBlockReaderFactory = bstream.BlockReaderFactoryFunc(BlockReaderFactory)
	bstream.GetBlockDecoder = bstream.BlockDecoderFunc(BlockDecoder)
	bstream.GetProtocolFirstStreamableBlock = 0
	bstream.GetProtocolGenesisBlock = 0
	bstream.GetBlockWriterHeaderLen = 10
}
