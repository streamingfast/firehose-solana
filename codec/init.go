package codec

import (
	"github.com/streamingfast/bstream"
	"time"
)

func init() {
	bstream.GetBlockWriterFactory = bstream.BlockWriterFactoryFunc(BlockWriterFactory)
	bstream.GetBlockReaderFactory = bstream.BlockReaderFactoryFunc(BlockReaderFactory)
	bstream.GetBlockDecoder = bstream.BlockDecoderFunc(BlockDecoder)
	bstream.GetProtocolFirstStreamableBlock = 0
	bstream.GetProtocolGenesisBlock = 0
	bstream.GetBlockWriterHeaderLen = 10

	bstream.GetBlockPayloadSetter = bstream.MemoryBlockPayloadSetter
	bstream.GetMemoizeMaxAge = 200 * 15 * time.Second

}
