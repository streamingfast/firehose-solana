module github.com/streamingfast/sf-solana

go 1.14

require (
	cloud.google.com/go/bigquery v1.10.0
	cloud.google.com/go/storage v1.10.0
	github.com/GeertJohan/go.rice v1.0.0
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/abourget/llerrgroup v0.2.0
	github.com/davecgh/go-spew v1.1.1
	github.com/dustin/go-humanize v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/graph-gophers/graphql-go v0.0.0-20201027172035-4c772c181653
	github.com/linkedin/goavro/v2 v2.10.0
	github.com/lorenzosaino/go-sysctl v0.1.1
	github.com/mholt/archiver/v3 v3.5.0
	github.com/mr-tron/base58 v1.2.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.8.1
	github.com/streamingfast/binary v0.0.0-20210811183519-94786c01e70d
	github.com/streamingfast/bstream v0.0.2-0.20210811181043-4c1920a7e3e3
	github.com/streamingfast/dauth v0.0.0-20210811181149-e8fd545948cc
	github.com/streamingfast/dbin v0.0.0-20210809205249-73d5eca35dc5
	github.com/streamingfast/derr v0.0.0-20210811180100-9138d738bcec
	github.com/streamingfast/dgraphql v0.0.2-0.20210811200910-e1966c29c473
	github.com/streamingfast/dgrpc v0.0.0-20210811180351-8646818518b2
	github.com/streamingfast/dlauncher v0.0.0-20210811194929-f06e488e63da
	github.com/streamingfast/dmetering v0.0.0-20210811181351-eef120cfb817
	github.com/streamingfast/dmetrics v0.0.0-20210811180524-8494aeb34447
	github.com/streamingfast/dstore v0.1.1-0.20210811180812-4db13e99cc22
	github.com/streamingfast/firehose v0.1.1-0.20210811195158-d4b116b4b447
	github.com/streamingfast/jsonpb v0.0.0-20210811021341-3670f0aa02d0
	github.com/streamingfast/kvdb v0.0.2-0.20210811194032-09bf862bd2e3
	github.com/streamingfast/logging v0.0.0-20210811175431-f3b44b61606a
	github.com/streamingfast/merger v0.0.3-0.20210811195536-1011c89f0a67
	github.com/streamingfast/node-manager v0.0.2-0.20210811195853-d6b519927636
	github.com/streamingfast/pbgo v0.0.6-0.20210811160400-7c146c2db8cc
	github.com/streamingfast/relayer v0.0.2-0.20210811200014-6e0e8bc2814f
	github.com/streamingfast/shutter v1.5.0
	github.com/streamingfast/solana-go v0.2.1-0.20210811184520-ab50363bdc52
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.17.0
	google.golang.org/api v0.53.0
	google.golang.org/grpc v1.39.1
	google.golang.org/protobuf v1.27.1 // indirect
	gorm.io/driver/bigquery v1.0.16
	gorm.io/gorm v1.20.13-0.20210223113524-940da051a756
)

replace github.com/graph-gophers/graphql-go => github.com/streamingfast/graphql-go v0.0.0-20210204202750-0e485a040a3c

replace github.com/ShinyTrinkets/overseer => github.com/dfuse-io/overseer v0.2.1-0.20191024193921-39856397cf3f
