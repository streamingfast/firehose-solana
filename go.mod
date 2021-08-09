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
	github.com/dfuse-io/bstream v0.0.2-0.20210218160250-ce6144227e87
	github.com/dfuse-io/dauth v0.0.0-20200601190857-60bc6a4b4665
	github.com/dfuse-io/dbin v0.0.0-20200406215642-ec7f22e794eb
	github.com/dfuse-io/derr v0.0.0-20201001203637-4dc9d8014152
	github.com/dfuse-io/dgraphql v0.0.2-0.20210209213422-0c203cf6bcfd
	github.com/dfuse-io/dgrpc v0.0.0-20210128133958-db1ca95920e4
	github.com/dfuse-io/dlauncher v0.0.0-20210224224510-27373532f11f
	github.com/dfuse-io/dmetering v0.0.0-20210112023524-c3ddadbc0d6a
	github.com/dfuse-io/dmetrics v0.0.0-20200508170817-3b8cb01fee68
	github.com/dfuse-io/dstore v0.1.1-0.20210204225142-e8106bdf280f
	github.com/dfuse-io/firehose v0.1.1-0.20210208183237-ab19a73e5842
	github.com/dfuse-io/jsonpb v0.0.0-20200602171045-28535c4016a2
	github.com/dfuse-io/kvdb v0.0.2-0.20201208184359-118334a9186e
	github.com/dfuse-io/logging v0.0.0-20210109005628-b97a57253f70
	github.com/dfuse-io/node-manager v0.0.2-0.20201211170554-49cc7e083f37
	github.com/dfuse-io/pbgo v0.0.6-0.20210125181705-b17235518132
	github.com/dfuse-io/relayer v0.0.2-0.20210202030730-e16ed570e7a9
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
	github.com/streamingfast/merger v0.0.3-0.20210809165038-14f85d21b69b // indirect
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
