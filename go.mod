module github.com/dfuse-io/dfuse-solana

go 1.14

require (
	github.com/GeertJohan/go.rice v1.0.0
	github.com/dfuse-io/bstream v0.0.2-0.20200714123252-e9115283f55f // indirect
	github.com/dfuse-io/derr v0.0.0-20201001203637-4dc9d8014152
	github.com/dfuse-io/dgraphql v0.0.1
	github.com/dfuse-io/dgrpc v0.0.0-20200417124327-c8f215bc4ce5
	github.com/dfuse-io/dlauncher v0.0.0-20200715193603-ea2a15e9e193
	github.com/dfuse-io/kvdb v0.0.0-20200508203924-c107cb0b2fa2
	github.com/dfuse-io/logging v0.0.0-20201023175426-d0173f8508dc
	github.com/dfuse-io/shutter v1.4.1-0.20200407040739-f908f9ab727f
	github.com/dfuse-io/solana-go v0.1.1-0.20201111152458-ef9111949e0c
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/graph-gophers/graphql-go v0.0.0-20201027172035-4c772c181653
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

replace github.com/graph-gophers/graphql-go => github.com/dfuse-io/graphql-go v0.0.0-20201111130519-96db37f31807
