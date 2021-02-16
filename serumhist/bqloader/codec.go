package bqloader

import (
	"github.com/linkedin/goavro/v2"
)

var (
	OrderCreatedCodec  *goavro.Codec
	OrderFilledCodec   *goavro.Codec
	TraderAccountCodec *goavro.Codec
)

func init() {
	var err error
	OrderCreatedCodec, err = goavro.NewCodec(``)
	if err != nil {
		//panic("unable to parse AVRO schema for OrderCreatedCodec")
	}
	OrderFilledCodec, err = goavro.NewCodec(``)
	if err != nil {
		//panic("unable to parse AVRO schema for OrderFilledCodec")
	}
	TraderAccountCodec, err = goavro.NewCodec(``)
	if err != nil {
		//panic("unable to parse AVRO schema for TraderAccountCodec")
	}
}
