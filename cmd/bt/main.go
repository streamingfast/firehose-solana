package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/bigtable"
	googleBigtable "cloud.google.com/go/bigtable"
	"github.com/mr-tron/base58"
	"github.com/streamingfast/firehose-solana/block/fetcher"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	client, err := googleBigtable.NewClient(ctx, "mainnet-beta", "solana-ledger")
	if err != nil {
		panic(err)
	}

	var logger, tracer = logging.PackageLogger("foo", "main")
	logging.InstantiateLoggers(logging.WithDefaultLevel(zap.DebugLevel))

	blockReader := fetcher.NewBigtableReader(client, 10, logger, tracer)

	table := client.Open("blocks")
	btRange := bigtable.NewRange(fmt.Sprintf("%016x", 140_000_000), "")

	err = table.ReadRows(ctx, btRange, func(row bigtable.Row) bool {
		block, _, err := blockReader.ProcessRow(row)
		if err != nil {
			panic(err)
		}

		//4xiPVLxJgpct8RKpVCV91x9aLYkfTwBayBw2JMFqw166a5atZgjSzYaSKqHVGpzGDLSgYYGdxdXeXFJkQCz6V7f5
		signatureBytes, err := base58.Decode("4sKtga1XVH6Q1g9JEXYwoTD8h9QU4DswKqsxu8JvPH2aMQT9ipKi6WCyYatH9DYRgMf6RCi8ZDPAmg1w1ZT1hJEs")
		if err != nil {
			fmt.Errorf("base58.Decode: %w", err)
		}

		for _, transaction := range block.Transactions {
			if bytes.Equal(transaction.Transaction.Signatures[0], signatureBytes) {
				fmt.Println("found")
				for _, balance := range transaction.Meta.PostTokenBalances {
					fmt.Println("Balance data :", balance)
				}
			}
		}

		return false
	})

	if err != nil {
		log.Fatalf("Could not read rows: %v", err)
	}

	//var logger, tracer = logging.PackageLogger("foo", "main")
	//logging.InstantiateLoggers(logging.WithDefaultLevel(zap.DebugLevel))
	//
	//blockReader := fetcher.NewBigtableReader(client, 10, logger, tracer)
	//
	//table := client.Open("blocks")
	//btRange := bigtable.NewRange(fmt.Sprintf("%016x", 11690699), "")
	//
	//err = table.ReadRows(ctx, btRange, func(row bigtable.Row) bool {
	//	block, _, err := blockReader.ProcessRow(row)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	//4xiPVLxJgpct8RKpVCV91x9aLYkfTwBayBw2JMFqw166a5atZgjSzYaSKqHVGpzGDLSgYYGdxdXeXFJkQCz6V7f5
	//	trxSign := "4xiPVLxJgpct8RKpVCV91x9aLYkfTwBayBw2JMFqw166a5atZgjSzYaSKqHVGpzGDLSgYYGdxdXeXFJkQCz6V7f5"
	//	signatureBytes, err := base58.Decode(trxSign)
	//	if err != nil {
	//		fmt.Errorf("base58.Decode: %w", err)
	//	}
	//
	//	for _, transaction := range block.Transactions {
	//		if bytes.Equal(transaction.Transaction.Signatures[0], signatureBytes) {
	//			fmt.Println("found:", trxSign)
	//		}
	//		if transaction.Meta.Err != nil {
	//			err := transaction.Meta.Err
	//			sign := base58.Encode(transaction.Transaction.Signatures[0])
	//			fmt.Println("err: ", sign, err.Err)
	//		}
	//	}
	//
	//	return false
	//})
	//
	//if err != nil {
	//	panic(err)
	//}
}
