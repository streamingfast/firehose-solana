package main

import (
	"fmt"
	"os"

	"github.com/dfuse-io/dfuse-solana/codec"
	pbcodec "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/codec/v1"
	"github.com/dfuse-io/jsonpb"
)

func main() {
	fl, err := os.Open("/home/abourget/dfuse/dfuse-solana/devel/mindreader-sync/dfuse-data/storage/one-blocks/0055455768-20201216T105128.2-b12ZkkWZ-QRBiNMNg.dbin")
	errCheck("mama", err)

	reader, err := codec.NewBlockReader(fl)
	errCheck("block", err)

	blk, err := reader.Read()
	errCheck("read", err)

	slot := blk.ToNative().(*pbcodec.Slot)

	cnt, err := jsonpb.MarshalIndentToString(slot, "  ")
	errCheck("marshal json", err)

	fmt.Println(string(cnt))
}

func errCheck(prefix string, err error) {
	if err != nil {
		fmt.Println(prefix, ":", err)
		os.Exit(1)
	}
}
