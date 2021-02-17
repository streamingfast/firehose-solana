package bqloader

import (
	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/dfuse-io/solana-go"
	"github.com/linkedin/goavro/v2"
)

var (
	CodecNewOrder      *goavro.Codec
	CodecOrderFilled   *goavro.Codec
	CodecTraderAccount *goavro.Codec
)

func init() {
	var err error
	CodecNewOrder, err = goavro.NewCodec(`{
		"namespace": "io.dfuse",
		"type": "record",
		"name": "",
		"fields": [],
	}`)
	if err != nil {
		//panic("unable to parse AVRO schema for CodecNewOrder")
	}
	CodecOrderFilled, err = goavro.NewCodec(`{
		"namespace": "io.dfuse",
		"type": "record",
		"name": "",
		"fields": [],
	}`)
	if err != nil {
		//panic("unable to parse AVRO schema for CodecOrderFilled")
	}
	CodecTraderAccount, err = goavro.NewCodec(`{
		"namespace": "io.dfuse",
		"type": "record",
		"name": "",
		"fields": [{"name": "account", "type": "string"},{"name": "trader", "type": "string"}],
	}`)
	if err != nil {
		//panic("unable to parse AVRO schema for CodecTraderAccount")
	}
}

func NewOrderToAvro(e *serumhist.NewOrder) map[string]interface{} {
	panic("implement me")
}

func FillEventToAvro(e *serumhist.FillEvent) map[string]interface{} {
	panic("implement me")
}

func TradingAccountToAvro(tradingAccount, trader solana.PublicKey) map[string]interface{} {
	return map[string]interface{}{
		"account": tradingAccount.String(),
		"trader":  trader.String(),
	}
}
