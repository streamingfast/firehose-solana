module github.com/dfuse-io/dfuse-solana

go 1.14

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.13.4 // indirect
	github.com/GeertJohan/go.rice v1.0.0
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/dfuse-io/binary v0.0.0-20210119182726-f245aa830ba8
	github.com/dfuse-io/bstream v0.0.2-0.20210118170643-057893cea2ef
	github.com/dfuse-io/dauth v0.0.0-20200601190857-60bc6a4b4665
	github.com/dfuse-io/dbin v0.0.0-20200406215642-ec7f22e794eb
	github.com/dfuse-io/derr v0.0.0-20201001203637-4dc9d8014152
	github.com/dfuse-io/dgraphql v0.0.2-0.20201204213310-1a60670e318b
	github.com/dfuse-io/dgrpc v0.0.0-20210116004319-046123544d11
	github.com/dfuse-io/dlauncher v0.0.0-20201215203933-750a56ede40d
	github.com/dfuse-io/dmetering v0.0.0-20210112023524-c3ddadbc0d6a
	github.com/dfuse-io/dmetrics v0.0.0-20200508170817-3b8cb01fee68
	github.com/dfuse-io/dstore v0.1.1-0.20201124190907-4b1585267864
	github.com/dfuse-io/firehose v0.1.1-0.20210118213034-5bdcff6a14a7
	github.com/dfuse-io/jsonpb v0.0.0-20200602171045-28535c4016a2
	github.com/dfuse-io/kvdb v0.0.2-0.20201208184359-118334a9186e
	github.com/dfuse-io/logging v0.0.0-20210109005628-b97a57253f70
	github.com/dfuse-io/merger v0.0.3-0.20210120192023-4faaf201eee9
	github.com/dfuse-io/node-manager v0.0.2-0.20201211170554-49cc7e083f37
	github.com/dfuse-io/pbgo v0.0.6-0.20210108215028-712d6889e94a
	github.com/dfuse-io/relayer v0.0.2-0.20201029161257-ec97edca50d7
	github.com/dfuse-io/shutter v1.4.1
	github.com/dfuse-io/solana-go v0.2.1-0.20210121235036-2fb4cfbfd6bc
	github.com/dustin/go-humanize v1.0.0
	github.com/golang/protobuf v1.4.2
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/graph-gophers/graphql-go v0.0.0-20201027172035-4c772c181653
	github.com/lorenzosaino/go-sysctl v0.1.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/teris-io/shortid v0.0.0-20201117134242-e59966efd125 // indirect
	github.com/test-go/testify v1.1.4
	github.com/tidwall/gjson v1.6.7 // indirect
	go.opencensus.io v0.22.5
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.25.0
	gotest.tools v2.2.0+incompatible
)

replace github.com/graph-gophers/graphql-go => github.com/dfuse-io/graphql-go v0.0.0-20201111130519-96db37f31807

replace github.com/ShinyTrinkets/overseer => github.com/maoueh/overseer v0.2.1-0.20191024193921-39856397cf3f
