module github.com/streamingfast/firehose-solana

go 1.15

require (
	cloud.google.com/go/bigtable v1.13.0
	cloud.google.com/go/storage v1.22.1
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/abourget/llerrgroup v0.2.0
	github.com/dustin/go-humanize v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/klauspost/compress v1.15.9
	github.com/lithammer/dedent v1.1.0
	github.com/lorenzosaino/go-sysctl v0.1.1
	github.com/manifoldco/promptui v0.8.0
	github.com/mholt/archiver/v3 v3.5.0
	github.com/mr-tron/base58 v1.2.0
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.10.1
	github.com/streamingfast/bstream v0.0.2-0.20220916182101-7a027bfdcffb
	github.com/streamingfast/cli v0.0.4-0.20220113202443-f7bcefa38f7e
	github.com/streamingfast/dauth v0.0.0-20220526210215-024098ade521
	github.com/streamingfast/derr v0.0.0-20220526184630-695c21740145
	github.com/streamingfast/dlauncher v0.0.0-20220909121534-7a9aa91dbb32
	github.com/streamingfast/dmetering v0.0.0-20220307162406-37261b4b3de9
	github.com/streamingfast/dmetrics v0.0.0-20220811180000-3e513057d17c
	github.com/streamingfast/dstore v0.1.1-0.20220921155016-7a52fdb3fe5f
	github.com/streamingfast/dtracing v0.0.0-20220305214756-b5c0e8699839 // indirect
	github.com/streamingfast/firehose v0.1.1-0.20220909121738-2f3bc007ea2b
	github.com/streamingfast/firehose-solana/types v0.0.0-20220929150934-05c197eb27df
	github.com/streamingfast/jsonpb v0.0.0-20210811021341-3670f0aa02d0
	github.com/streamingfast/kvdb v0.0.2-0.20210811194032-09bf862bd2e3
	github.com/streamingfast/logging v0.0.0-20220813175024-b4fbb0e893df
	github.com/streamingfast/merger v0.0.3-0.20220909122033-9ca15beb25f5
	github.com/streamingfast/node-manager v0.0.2-0.20220912235129-6c08463b0c01
	github.com/streamingfast/pbgo v0.0.6-0.20220630154121-2e8bba36234e
	github.com/streamingfast/relayer v0.0.2-0.20220909122435-e67fbc964fd9
	github.com/streamingfast/sf-tools v0.0.0-20220830151952-184d6e9a6bb9
	github.com/streamingfast/shutter v1.5.0
	github.com/streamingfast/solana-go v0.5.1-0.20220502224452-432fbe84aee8
	github.com/streamingfast/substreams v0.0.21-0.20220929191348-9be622327416
	github.com/stretchr/testify v1.8.0
	github.com/teris-io/shortid v0.0.0-20201117134242-e59966efd125 // indirect
	github.com/test-go/testify v1.1.4
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292
	golang.org/x/tools v0.1.10 // indirect
	google.golang.org/api v0.91.0
	google.golang.org/grpc v1.49.0
	google.golang.org/protobuf v1.28.0
)

replace github.com/graph-gophers/graphql-go => github.com/streamingfast/graphql-go v0.0.0-20210204202750-0e485a040a3c

replace github.com/ShinyTrinkets/overseer => github.com/dfuse-io/overseer v0.2.1-0.20191024193921-39856397cf3f
