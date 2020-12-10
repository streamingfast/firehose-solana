package cli

const (
	//Protocol               pbbstream.Protocol = pbbstream.Protocol_EOS //todo
	TrxDBDSN               string = "badger://{dfuse-data-dir}/storage/trxdb"   //%s will be replaced by `<data-dir>`
	FluxDSN                string = "badger://{dfuse-data-dir}/storage/statedb" //%s will be replaced by `<data-dir>/<flux-data-dir>
	MergedBlocksStoreURL   string = "file://{dfuse-data-dir}/storage/merged-blocks"
	FilteredBlocksStoreURL string = "file://{dfuse-data-dir}/storage/filtered-merged-blocks"
	IndicesStoreURL        string = "file://{dfuse-data-dir}/storage/indexes"
	OneBlockStoreURL       string = "file://{dfuse-data-dir}/storage/one-blocks"
	SnapshotsURL           string = "file://{dfuse-data-dir}/storage/snapshots"
	DmeshDSN               string = "local://"
	DmeshServiceVersion    string = "v1"
	NetworkID              string = "sol-local"

	APIProxyHTTPListenAddr  string = ":8080"
	DashboardHTTPListenAddr string = ":8081"
	MetricsListenAddr       string = ":9102"

	MinerNodeHTTPServingAddr      string = ":13001"
	MindreaderNodeHTTPServingAddr string = ":13002"
	MindreaderNodeGRPCAddr        string = ":13003"
	PeeringNodeHTTPServingAddr    string = ":13004"
	BackupNodeHTTPServingAddr     string = ":13005"
	RelayerServingAddr            string = ":13006"
	MergerServingAddr             string = ":13007"
	AbiServingAddr                string = ":13008"
	BlockmetaServingAddr          string = ":13009"
	ArchiveServingAddr            string = ":13010"
	ArchiveHTTPServingAddr        string = ":13011"
	LiveServingAddr               string = ":13012"
	RouterServingAddr             string = ":13013"
	RouterHTTPServingAddr         string = ":13014"
	TrxDBHTTPServingAddr          string = ":13015"
	IndexerServingAddr            string = ":13016"
	IndexerHTTPServingAddr        string = ":13017"
	DgraphqlHTTPServingAddr       string = ":13018"
	DgraphqlGRPCServingAddr       string = ":13019"
	ForkResolverServingAddr       string = ":13020"
	ForkResolverHTTPServingAddr   string = ":13021"
	SolqHTTPServingAddr           string = ":13022"
	DashboardGRPCServingAddr      string = ":13023"
	FilteringRelayerServingAddr   string = ":13024"
	TokenmetaGRPCServingAddr      string = ":13025"

	// Solana node instance port definitions
	MinerNodeRPCPort      string = "13100"
	MinerNodeGossipPort   string = "13101"
	MinerNodeP2PPortStart string = "13102"
	MinerNodeP2PPortEnd   string = "13199"

	MindreaderNodeRPCPort      string = "13200"
	MindreaderNodeGossipPort   string = "13201"
	MindreaderNodeP2PPortStart string = "13202"
	MindreaderNodeP2PPortEnd   string = "13299"

	PeeringNodeRPCPort      string = "13300"
	PeeringNodeGossipPort   string = "13301"
	PeeringNodeP2PPortStart string = "13302"
	PeeringNodeP2PPortEnd   string = "13399"

	BackupNodeRPCPort      string = "13400"
	BackupNodeGossipPort   string = "13401"
	BackupNodeP2PPortStart string = "13402"
	BackupNodeP2PPortEnd   string = "13499"
)
