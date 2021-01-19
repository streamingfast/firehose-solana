module github.com/dfuse-io/dfuse-solana

go 1.14

require (
	github.com/GeertJohan/go.rice v1.0.0
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/dfuse-io/binary v0.0.0-20210119182726-f245aa830ba8
	github.com/dfuse-io/bstream v0.0.2-0.20210118170643-057893cea2ef
	github.com/dfuse-io/dauth v0.0.0-20200529171443-21c0e2d262c2
	github.com/dfuse-io/dbin v0.0.0-20200406215642-ec7f22e794eb
	github.com/dfuse-io/derr v0.0.0-20201001203637-4dc9d8014152
	github.com/dfuse-io/dgraphql v0.0.2-0.20201204213310-1a60670e318b
	github.com/dfuse-io/dgrpc v0.0.0-20201215171222-11bde2006cf9
	github.com/dfuse-io/dlauncher v0.0.0-20201215203933-750a56ede40d
	github.com/dfuse-io/dmetrics v0.0.0-20200508152325-93e7e9d576bb
	github.com/dfuse-io/dstore v0.1.1-0.20201124190907-4b1585267864
	github.com/dfuse-io/jsonpb v0.0.0-20200406211248-c5cf83f0e0c0
	github.com/dfuse-io/kvdb v0.0.2-0.20201208184359-118334a9186e
	github.com/dfuse-io/logging v0.0.0-20201125153217-f29c382faa42
	github.com/dfuse-io/merger v0.0.3-0.20210104194844-46a615b93bef
	github.com/dfuse-io/node-manager v0.0.2-0.20201211170554-49cc7e083f37
	github.com/dfuse-io/pbgo v0.0.6-0.20210108215028-712d6889e94a
	github.com/dfuse-io/relayer v0.0.2-0.20201029161257-ec97edca50d7
	github.com/dfuse-io/shutter v1.4.1-0.20200407040739-f908f9ab727f
	github.com/dfuse-io/solana-go v0.2.1-0.20210119190242-57bebed0dae0
	github.com/dustin/go-humanize v1.0.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/graph-gophers/graphql-go v0.0.0-20201027172035-4c772c181653
	github.com/lorenzosaino/go-sysctl v0.1.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/test-go/testify v1.1.4
	go.opencensus.io v0.22.4
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.25.0
	gotest.tools v2.2.0+incompatible
)

replace github.com/graph-gophers/graphql-go => github.com/dfuse-io/graphql-go v0.0.0-20201111130519-96db37f31807

replace github.com/ShinyTrinkets/overseer => github.com/maoueh/overseer v0.2.1-0.20191024193921-39856397cf3f
