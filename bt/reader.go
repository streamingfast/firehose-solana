package bt

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os/exec"
	"strings"

	"cloud.google.com/go/bigtable"
	"github.com/golang/protobuf/proto"
	"github.com/klauspost/compress/zstd"
	pbsolv1 "github.com/streamingfast/sf-solana/types/pb/sf/solana/type/v1"
	"go.uber.org/zap"
)

type RowType string

const (
	RowTypeProto RowType = "proto"
	RowTypeBin   RowType = "bin"
)

func ExplodeRow(row bigtable.Row) (*big.Int, RowType, []byte) {
	el := row["x"][0]
	var rowType RowType
	if strings.HasSuffix(el.Column, "proto") {
		rowType = RowTypeProto
	} else {
		rowType = RowTypeBin
	}
	blockNum, _ := new(big.Int).SetString(el.Row, 16)
	return blockNum, rowType, el.Value
}

func ProcessRow(row bigtable.Row, zlogger *zap.Logger) (*pbsolv1.Block, error) {
	blockNum, rowType, rowCnt := ExplodeRow(row)
	zlogger.Debug("found bigtable row", zap.Stringer("blk_num", blockNum), zap.Int("uncompressed_length", len(rowCnt)))
	var cnt []byte
	var err error

	switch rowType {
	case RowTypeBin:
		cnt, err = externalBinToProto(rowCnt, "solana-bigtable-decoder", "--hex")
		if err != nil {
			return nil, fmt.Errorf("unable get external bin %s: %w", blockNum.String(), err)
		}
	default:
		cnt, err = Decompress(rowCnt)
		if err != nil {
			return nil, fmt.Errorf("unable to decompress block %s (uncompresse length %d): %w", blockNum.String(), len(rowCnt), err)
		}

	}
	zlogger.Debug("found bigtable row", zap.Stringer("blk_num", blockNum),
		zap.String("key", row.Key()),
		zap.Int("compressed_length", len(rowCnt)),
		zap.Int("uncompressed_length", len(cnt)),
		zap.String("row_type", string(rowType)),
	)

	blk := &pbsolv1.Block{}
	if err := proto.Unmarshal(cnt, blk); err != nil {
		return nil, fmt.Errorf("unable to unmarshall confirmed block: %w", err)
	}
	blk.Slot = blockNum.Uint64()

	// horrible tweaks
	switch blk.Blockhash {
	case "Goi3t9JjgDkyULZbM2TzE5QqHP1fPeMcHNaXNFBCBv1v":
		zlog.Warn("applying horrible tweak to block Goi3t9JjgDkyULZbM2TzE5QqHP1fPeMcHNaXNFBCBv1v")
		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
			blk.PreviousBlockhash = "HQEr9qcbUVBt2okfu755FdJvJrPYTSpzzmmyeWTj5oau"
		}
	case "6UFQveZ94DUKGbcLFoyayn1QwthVfD3ZqvrM2916pHCR":
		zlog.Warn("applying horrible tweak to block 63,072,071")
		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
			blk.PreviousBlockhash = "7cLQx2cZvyKbGoMuutXEZ3peg3D21D5qbX19T5V1XEiK"
		}
	case "Fqbm7QvCTYnToXWcCw6nbkWhMmXx2Nv91LsXBrKraB43":
		zlog.Warn("applying horrible tweak to block 53,135,959")
		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
			blk.PreviousBlockhash = "RfXUrekgajPSb1R4CGFJWNaHTnB6p53Tzert4gouj2u"
		}
	case "ABp9G2NaPzM6kQbeyZYCYgdzL8JN9AxSSbCQG2X1K9UF":
		zlog.Warn("applying horrible tweak to block 46,223,993")
		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
			blk.PreviousBlockhash = "9F2C7TGqUpFu6krd8vQbUv64BskrneBSgY7U2QfrGx96"
		}
	}

	return blk, nil
}

func externalBinToProto(in []byte, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	inString := hex.EncodeToString(in)

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, inString)
	}()

	outCntHex, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	outHex := string(outCntHex)
	if strings.HasPrefix(outHex, "0x") {
		outHex = outHex[2:]
	}
	outHex = strings.TrimRight(outHex, "\n")
	return hex.DecodeString(outHex)
}

func Decompress(in []byte) (out []byte, err error) {
	switch in[0] {
	case 0:
		zlog.Debug("no compression found")
		out = in[4:]
	case 1:
		zlog.Debug("bzip2 compression")
		// bzip2
		out, err = ioutil.ReadAll(bzip2.NewReader(bytes.NewBuffer(in[4:])))
		if err != nil {
			return nil, fmt.Errorf("bzip2 decompress: %w", err)
		}
	case 2:
		zlog.Debug("gzip compression")
		// gzip
		reader, err := gzip.NewReader(bytes.NewBuffer(in[4:]))
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		out, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("gzip decompress: %w", err)
		}
	case 3:
		zlog.Debug("zstd compression")
		// zstd
		var dec *zstd.Decoder
		dec, err = zstd.NewReader(nil)
		if err != nil {
			return nil, fmt.Errorf("zstd reader: %w", err)
		}
		out, err = dec.DecodeAll(in[4:], out)
		if err != nil {
			return nil, fmt.Errorf("zstd decompress: %w", err)

		}
	default:
		return nil, fmt.Errorf("unsupported compression scheme for a block %d", in[0])
	}
	return
}
