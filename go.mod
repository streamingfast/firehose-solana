module github.com/dfuse-io/dfuse-solana

go 1.14

require (
	cloud.google.com/go/storage v1.8.0
	github.com/GeertJohan/go.rice v1.0.0
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/abourget/llerrgroup v0.2.0
	github.com/dfuse-io/binary v0.0.0-20210125232659-d265783d8b7c
	github.com/dfuse-io/bstream v0.0.2-0.20210203203654-afe75df13683
	github.com/dfuse-io/dauth v0.0.0-20200601190857-60bc6a4b4665
	github.com/dfuse-io/dbin v0.0.0-20200406215642-ec7f22e794eb
	github.com/dfuse-io/derr v0.0.0-20201001203637-4dc9d8014152
	github.com/dfuse-io/dgraphql v0.0.2-0.20210128181646-b7f03ff95e0a
	github.com/dfuse-io/dgrpc v0.0.0-20210128133958-db1ca95920e4
	github.com/dfuse-io/dlauncher v0.0.0-20201215203933-750a56ede40d
	github.com/dfuse-io/dmetering v0.0.0-20210112023524-c3ddadbc0d6a
	github.com/dfuse-io/dmetrics v0.0.0-20200508170817-3b8cb01fee68
	github.com/dfuse-io/dstore v0.1.1-0.20210204225142-e8106bdf280f
	github.com/dfuse-io/firehose v0.1.1-0.20210203173222-44009f1096c5
	github.com/dfuse-io/jsonpb v0.0.0-20200602171045-28535c4016a2
	github.com/dfuse-io/kvdb v0.0.2-0.20201208184359-118334a9186e
	github.com/dfuse-io/logging v0.0.0-20210109005628-b97a57253f70
	github.com/dfuse-io/merger v0.0.3-0.20210120192023-4faaf201eee9
	github.com/dfuse-io/node-manager v0.0.2-0.20201211170554-49cc7e083f37
	github.com/dfuse-io/pbgo v0.0.6-0.20210125181705-b17235518132
	github.com/dfuse-io/relayer v0.0.2-0.20210202030730-e16ed570e7a9
	github.com/dfuse-io/shutter v1.4.1
	github.com/dfuse-io/solana-go v0.2.1-0.20210126234342-be9990a71471
	github.com/dustin/go-humanize v1.0.0
	github.com/golang/protobuf v1.4.2
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/graph-gophers/graphql-go v0.0.0-20201027172035-4c772c181653
	github.com/lorenzosaino/go-sysctl v0.1.1
	github.com/mholt/archiver/v3 v3.5.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/test-go/testify v1.1.4
	go.uber.org/zap v1.16.0
	google.golang.org/api v0.29.0
	google.golang.org/grpc v1.29.1
)

replace github.com/graph-gophers/graphql-go => github.com/dfuse-io/graphql-go v0.0.0-20201111130519-96db37f31807

replace github.com/ShinyTrinkets/overseer => github.com/maoueh/overseer v0.2.1-0.20191024193921-39856397cf3f
