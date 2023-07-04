package cli

const (
	MergedBlocksStoreURL string = "file://{data-dir}/storage/merged-blocks"
	OneBlockStoreURL     string = "file://{data-dir}/storage/one-blocks"
	ForkedBlocksStoreURL string = "file://{data-dir}/storage/forked-blocks"
	BlocksCacheDirectory string = "{data-dir}/blocks-cache"
	NetworkID            string = "development"
	SFNetworkID          string = "sol-local"

	APIProxyHTTPListenAddr  string = ":8080"
	DashboardHTTPListenAddr string = ":8081"
	MetricsListenAddr       string = ":9102"

	SubstreamsTier1GRPCServingAddr string = ":13044"
	SubstreamsTier2GRPCServingAddr string = ":13045"

	MinerNodeHTTPServingAddr   string = ":14001"
	ReaderNodeHTTPServingAddr  string = ":14002"
	ReaderNodeGRPCAddr         string = ":14003"
	PeeringNodeHTTPServingAddr string = ":14004"
	RelayerServingAddr         string = ":14006"
	MergerServingAddr          string = ":14007"
	BlockmetaServingAddr       string = ":14009"
	FirehoseGRPCServingAddr    string = ":14026"

	// Solana node instance port definitions
	MinerNodeRPCPort      string = "14100"
	MinerNodeRPCWSPort    string = "14101"
	MinerNodeGossipPort   string = "14110"
	MinerNodeP2PPortStart string = "14111"
	MinerNodeP2PPortEnd   string = "14199"

	ReaderNodeRPCPort      string = "14200"
	ReaderNodeRPCWSPort    string = "14201"
	ReaderNodeGossipPort   string = "14210"
	ReaderNodeP2PPortStart string = "14211"
	ReaderNodeP2PPortEnd   string = "14299"
)
