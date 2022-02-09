module github.com/streamingfast/sf-solana

go 1.15

require (
	cloud.google.com/go/storage v1.10.0
	github.com/GeertJohan/go.rice v1.0.0
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/abourget/llerrgroup v0.2.0
	github.com/dustin/go-humanize v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/graph-gophers/graphql-go v0.0.0-20201027172035-4c772c181653
	github.com/lorenzosaino/go-sysctl v0.1.1
	github.com/mholt/archiver/v3 v3.5.0
	github.com/mr-tron/base58 v1.2.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.9.0
	github.com/streamingfast/bstream v0.0.2-0.20220131215301-c3b4f674a5b2
	github.com/streamingfast/dauth v0.0.0-20210812020920-1c83ba29add1
	github.com/streamingfast/dbin v0.0.0-20210809205249-73d5eca35dc5
	github.com/streamingfast/derr v0.0.0-20210811180100-9138d738bcec
	github.com/streamingfast/dgraphql v0.0.2-0.20211210154505-08e159e66cfc
	github.com/streamingfast/dgrpc v0.0.0-20211210152421-f8cec68e0383
	github.com/streamingfast/dlauncher v0.0.0-20211210162313-cf4aa5fc4878
	github.com/streamingfast/dmetering v0.0.0-20210811181351-eef120cfb817
	github.com/streamingfast/dmetrics v0.0.0-20210811180524-8494aeb34447
	github.com/streamingfast/dstore v0.1.1-0.20220203133825-30eb2f9c5cd3
	github.com/streamingfast/firehose v0.1.1-0.20220126192116-2a4e7fd327e5
	github.com/streamingfast/jsonpb v0.0.0-20210811021341-3670f0aa02d0
	github.com/streamingfast/kvdb v0.0.2-0.20210811194032-09bf862bd2e3
	github.com/streamingfast/logging v0.0.0-20211201142855-8f6ea4c04c74
	github.com/streamingfast/merger v0.0.3-0.20220203135657-6583cdf4c763
	github.com/streamingfast/node-manager v0.0.2-0.20211029201743-0b82ab7f9de4
	github.com/streamingfast/pbgo v0.0.6-0.20220104194237-6534a2f6320b
	github.com/streamingfast/relayer v0.0.2-0.20220120224524-84b9578c9323
	github.com/streamingfast/shutter v1.5.0
	github.com/streamingfast/solana-go v0.3.1-0.20211123130545-cec9725a7d7a
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.1
	google.golang.org/api v0.59.0
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/graph-gophers/graphql-go => github.com/streamingfast/graphql-go v0.0.0-20210204202750-0e485a040a3c

replace github.com/ShinyTrinkets/overseer => github.com/dfuse-io/overseer v0.2.1-0.20191024193921-39856397cf3f
