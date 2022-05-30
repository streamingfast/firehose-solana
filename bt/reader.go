package bt

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"strings"

	msgpack "github.com/Xuanwo/serde-msgpack-go"

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
	if cnt, err = Decompress(rowCnt); err != nil {
		return nil, fmt.Errorf("unable to decompress block %s (uncompresse length %d): %w", blockNum.String(), len(rowCnt), err)
	}
	zlogger.Debug("found bigtable row", zap.Stringer("blk_num", blockNum),
		zap.Int("uncompressed_length", len(rowCnt)),
		zap.Int("compressed_length", len(cnt)),
		zap.String("row_type", string(rowType)),
	)
	blk := &pbsolv1.Block{}

	if rowType == RowTypeBin {

		x := serdeStructFieldVisitor_StoredConfirmedBlock{}
		err = msgpack.DeserializeFromBytes(cnt, &x)
		if err != nil {
			log.Fatalf("msgpack DeserializeFromBytes: %v", err)
		}
		log.Printf("%#+v", x)

		buf := bytes.NewBuffer(cnt)
		decoder := gob.NewDecoder(buf)
		if err := decoder.Decode(&blk); err != nil {
			return nil, fmt.Errorf("unable to bin decode confirmed block: %w", err)
		}
		//return nil, fmt.Errorf("unable to decode row of type binary")
	}

	if err := proto.Unmarshal(cnt, blk); err != nil {
		return nil, fmt.Errorf("unable to unmarshall confirmed block: %w", err)
	}
	blk.Slot = blockNum.Uint64()
	return blk, nil
}

func Decompress(in []byte) (out []byte, err error) {
	switch in[0] {
	case 0:
		out = in[4:]
	case 1:
		// bzip2
		out, err = ioutil.ReadAll(bzip2.NewReader(bytes.NewBuffer(in[4:])))
		if err != nil {
			return nil, fmt.Errorf("bzip2 decompress: %w", err)
		}
	case 2:
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
