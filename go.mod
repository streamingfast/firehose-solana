module github.com/streamingfast/firehose-solana

go 1.15

require (
	cloud.google.com/go/bigtable v1.13.0
	cloud.google.com/go/storage v1.23.0
	github.com/Azure/go-autorest/autorest/adal v0.9.20 // indirect
	github.com/Azure/go-autorest/autorest/mocks v0.4.2 // indirect
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/abourget/llerrgroup v0.2.0
	github.com/dustin/go-humanize v1.0.0
	github.com/golang-jwt/jwt/v4 v4.2.0 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/klauspost/compress v1.15.9
	github.com/lithammer/dedent v1.1.0
	github.com/lorenzosaino/go-sysctl v0.1.1
	github.com/manifoldco/promptui v0.8.0
	github.com/mholt/archiver/v3 v3.5.0
	github.com/mr-tron/base58 v1.2.0
	github.com/onsi/ginkgo/v2 v2.1.4 // indirect
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.10.1
	github.com/streamingfast/bstream v0.0.2-0.20230120170832-6389e61e9d54
	github.com/streamingfast/cli v0.0.4-0.20220630165922-bc58c6666fc8
	github.com/streamingfast/dauth v0.0.0-20221027185237-b209f25fa3ff
	github.com/streamingfast/dbin v0.9.1-0.20220513054835-1abebbb944ad // indirect
	github.com/streamingfast/derr v0.0.0-20221125175206-82e01d420d45
	github.com/streamingfast/dgrpc v0.0.0-20230113212008-1898f17e0ac7
	github.com/streamingfast/dlauncher v0.0.0-20220909121534-7a9aa91dbb32
	github.com/streamingfast/dmetering v0.0.0-20220307162406-37261b4b3de9
	github.com/streamingfast/dmetrics v0.0.0-20221129121022-a1733eca1981
	github.com/streamingfast/dstore v0.1.1-0.20230117140537-2c870a0d4523
	github.com/streamingfast/dtracing v0.0.0-20220305214756-b5c0e8699839 // indirect
	github.com/streamingfast/firehose v0.1.1-0.20221101130227-3a0b1980aa0b
	github.com/streamingfast/firehose-solana/types v0.0.0-20230126211203-a2d17ce9f8b9
	github.com/streamingfast/jsonpb v0.0.0-20210811021341-3670f0aa02d0
	github.com/streamingfast/kvdb v0.0.2-0.20210811194032-09bf862bd2e3
	github.com/streamingfast/logging v0.0.0-20220813175024-b4fbb0e893df
	github.com/streamingfast/merger v0.0.3-0.20221123202507-445dfd357868
	github.com/streamingfast/node-manager v0.0.2-0.20221115101723-d9823ffd7ad5
	github.com/streamingfast/pbgo v0.0.6-0.20221020131607-255008258d28
	github.com/streamingfast/relayer v0.0.2-0.20220909122435-e67fbc964fd9
	github.com/streamingfast/sf-tools v0.0.0-20221129171534-a0708b599ce5
	github.com/streamingfast/shutter v1.5.0
	github.com/streamingfast/solana-go v0.5.1-0.20220502224452-432fbe84aee8
	github.com/streamingfast/substreams v0.1.0
	github.com/stretchr/testify v1.8.0
	github.com/teris-io/shortid v0.0.0-20201117134242-e59966efd125 // indirect
	github.com/test-go/testify v1.1.4
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd
	google.golang.org/api v0.99.0
	google.golang.org/grpc v1.50.1
	google.golang.org/protobuf v1.28.1
)

replace github.com/graph-gophers/graphql-go => github.com/streamingfast/graphql-go v0.0.0-20210204202750-0e485a040a3c

replace github.com/bytecodealliance/wasmtime-go => github.com/streamingfast/wasmtime-go v0.0.0-20221110162800-03b0c10e9b6a

replace github.com/ShinyTrinkets/overseer => github.com/dfuse-io/overseer v0.2.1-0.20191024193921-39856397cf3f
