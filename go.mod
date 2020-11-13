module github.com/dfuse-io/dfuse-solana

go 1.14

require (
	github.com/GeertJohan/go.rice v1.0.0
	github.com/dfuse-io/derr v0.0.0-20201001203637-4dc9d8014152
	github.com/dfuse-io/dgrpc v0.0.0-20200417124327-c8f215bc4ce5
	github.com/dfuse-io/dlauncher v0.0.0-20200715193603-ea2a15e9e193
	github.com/dfuse-io/kvdb v0.0.0-20200508203924-c107cb0b2fa2
	github.com/dfuse-io/logging v0.0.0-20201110202154-26697de88c79
	github.com/dfuse-io/shutter v1.4.1-0.20200407040739-f908f9ab727f
	github.com/dfuse-io/solana-go v0.1.1-0.20201113175514-e22f93385830
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/graph-gophers/graphql-go v0.0.0-20201027172035-4c772c181653
	github.com/lunixbochs/struc v0.0.0-20200707160740-784aaebc1d40 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
)

replace github.com/graph-gophers/graphql-go => github.com/dfuse-io/graphql-go v0.0.0-20201111130519-96db37f31807
