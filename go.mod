module github.com/dfuse-io/dfuse-solana

go 1.14

require (
	cloud.google.com/go/bigquery v1.10.0
	cloud.google.com/go/storage v1.10.0
	github.com/GeertJohan/go.rice v1.0.0
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/abourget/llerrgroup v0.2.0
	github.com/davecgh/go-spew v1.1.1
	github.com/dfuse-io/binary v0.0.0-20210216024852-4ae6830a495d
	github.com/dfuse-io/bstream v0.0.2-0.20210810055526-1c2b722f0cf6
	github.com/dfuse-io/dgrpc v0.0.0-20210424033943-10e04dd5b19c
	github.com/dfuse-io/dmetrics v0.0.0-20200508170817-3b8cb01fee68
	github.com/dfuse-io/dstore v0.1.1-0.20210507180120-88a95674809f
	github.com/dfuse-io/jsonpb v0.0.0-20200602171045-28535c4016a2
	github.com/dfuse-io/kvdb v0.0.2-0.20201208184359-118334a9186e
	github.com/dfuse-io/logging v0.0.0-20210109005628-b97a57253f70
	github.com/dfuse-io/pbgo v0.0.6-0.20210429181308-d54fc7723ad3
	github.com/dfuse-io/shutter v1.4.1
	github.com/dfuse-io/solana-go v0.2.1-0.20210218235942-214d7803f326
	github.com/dustin/go-humanize v1.0.0
	github.com/facebookgo/ensure v0.0.0-20200202191622-63f1cf65ac4c // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20200203212716-c811ad88dec4 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/graph-gophers/graphql-go v0.0.0-20201027172035-4c772c181653
	github.com/linkedin/goavro/v2 v2.8.5
	github.com/lorenzosaino/go-sysctl v0.1.1
	github.com/mholt/archiver/v3 v3.5.0
	github.com/mr-tron/base58 v1.2.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/streamingfast/dauth v0.0.0-20210809192433-4c758fd333ac
	github.com/streamingfast/dbin v0.0.0-20210809205249-73d5eca35dc5
	github.com/streamingfast/derr v0.0.0-20210810022442-32249850a4fb
	github.com/streamingfast/dgraphql v0.0.2-0.20210809194855-abdc276216a4
	github.com/streamingfast/dlauncher v0.0.0-20210809203103-fe27d137acc7
	github.com/streamingfast/dmetering v0.0.0-20210809193048-81d008c90843
	github.com/streamingfast/firehose v0.1.1-0.20210809193802-776cf9f9942e
	github.com/streamingfast/merger v0.0.3-0.20210810060447-9dbf5a0ab9e9
	github.com/streamingfast/node-manager v0.0.2-0.20210810060535-08c7a3f7fa5a
	github.com/streamingfast/relayer v0.0.2-0.20210809195208-c686bf91e083
	github.com/stretchr/testify v1.6.1
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c // indirect
	github.com/test-go/testify v1.1.4
	github.com/tidwall/gjson v1.6.7 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/zap v1.16.0
	google.golang.org/api v0.36.0
	google.golang.org/grpc v1.34.1
	google.golang.org/grpc/examples v0.0.0-20210223174733-dabedfb38b74 // indirect
	gorm.io/driver/bigquery v1.0.16
	gorm.io/gorm v1.20.13-0.20210223113524-940da051a756
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/graph-gophers/graphql-go => github.com/dfuse-io/graphql-go v0.0.0-20210204202750-0e485a040a3c

replace github.com/ShinyTrinkets/overseer => github.com/maoueh/overseer v0.2.1-0.20191024193921-39856397cf3f
