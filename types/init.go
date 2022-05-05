package types

import (
	"fmt"
	"io"
	"time"

	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"

	"github.com/streamingfast/bstream"
)

var CurrentMode = "standard"

func IsSfSolAugmented() bool {
	return CurrentMode == "augmented"
}

func SetupSfSolAugmented() {
	bstream.GetBlockDecoder = bstream.BlockDecoderFunc(PBSolBlockDecoder)
	CurrentMode = "augmented"
}

func init() {
	bstream.GetBlockReaderFactory = bstream.BlockReaderFactoryFunc(blockReaderFactory)
	bstream.GetBlockWriterFactory = bstream.BlockWriterFactoryFunc(blockWriterFactory)
	bstream.GetProtocolFirstStreamableBlock = 0
	bstream.GetBlockWriterHeaderLen = 10
	bstream.GetBlockPayloadSetter = bstream.MemoryBlockPayloadSetter
	bstream.GetMemoizeMaxAge = 200 * 15 * time.Second
	bstream.GetBlockDecoder = bstream.BlockDecoderFunc(PBSolanaBlockDecoder)
}

func blockReaderFactory(reader io.Reader) (bstream.BlockReader, error) {
	return bstream.NewDBinBlockReader(reader, func(contentType string, version int32) error {
		protocol := pbbstream.Protocol(pbbstream.Protocol_value[contentType])
		if protocol != pbbstream.Protocol_SOLANA && version != 1 {
			return fmt.Errorf("reader only knows about %s block kind at version 1, got %s at version %d", protocol, contentType, version)
		}

		return nil
	})
}

func blockWriterFactory(writer io.Writer) (bstream.BlockWriter, error) {
	return bstream.NewDBinBlockWriter(writer, "SOL", 1)
}
