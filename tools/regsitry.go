package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/dfuse-io/dfuse-solana/registry"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/solana-go/programs/serum"
	solrpc "github.com/dfuse-io/solana-go/rpc"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{Use: "registry", Short: "Registry "}
var fetchRegistryCmd = &cobra.Command{Use: "fetch", Short: "Fetch"}
var marketsFetchCmd = &cobra.Command{
	Use:   "markets {old-market.jsonl} {new-market.jsonl}",
	Short: "Retrieve Serum Markets",
	Long:  "Retrieve Serum Markets",
	Args:  cobra.ExactArgs(2),
	RunE:  fetchMarketsE,
}

func init() {
	Cmd.AddCommand(registryCmd)
	fetchRegistryCmd.AddCommand(marketsFetchCmd)
	registryCmd.AddCommand(fetchRegistryCmd)

	registryCmd.PersistentFlags().String("rpc", "https://solana-api.projectserum.com", "RPC URL")
	marketsFetchCmd.PersistentFlags().String("store", "gs://staging.dfuseio-global.appspot.com/sol-markets", "Market st")

}

func fetchMarketsE(cmd *cobra.Command, args []string) (err error) {
	// todo this should be an argument
	oldFilepath := fmt.Sprintf("%s/%s", viper.GetString("store"), args[0])
	newFilename := args[1]

	ctx := cmd.Context()
	client := solrpc.NewClient(viper.GetString("rpc"))
	fmt.Printf("Retrieving outdated markets: %s\n", oldFilepath)
	oldMarkets, err := readKnownMarkets(ctx, oldFilepath)
	if err != nil {
		return fmt.Errorf("unable to retrieve known markets: %w", err)
	}
	fmt.Printf("Markets V1 contains %d markets\n", len(oldMarkets))

	fmt.Printf("Retrieving on-chain markets for program: %s\n", serum.PROGRAM_ID.String())
	accounts, err := client.GetProgramAccounts(
		ctx,
		serum.PROGRAM_ID,
		&solrpc.GetProgramAccountsOpts{
			Filters: []solrpc.RPCFilter{
				{
					Memcmp: &solrpc.RPCFilterMemcmp{
						Offset: 5,
						Bytes: []byte{
							0x03,
						},
					},
				},
			},
		},
	)
	fmt.Printf("Found %d on chain markets for program %s\n", len(accounts), serum.PROGRAM_ID.String())

	f, err := os.Create(filepath.Join("/tmp", "out.jsonl"))
	if err != nil {
		return fmt.Errorf("unable to open local file write: %w", err)
	}
	defer f.Close()

	for _, acc := range accounts {

		market := &serum.MarketV2{}
		err := market.Decode(acc.Account.Data)
		if err != nil {
			fmt.Printf("Skipping account %s unable to decode market\n", acc.Pubkey.String())
			continue
		}

		marketRegistry := &registry.Market{}
		if m, found := oldMarkets[acc.Pubkey.String()]; found {
			marketRegistry = m
		}

		marketRegistry.Address = acc.Pubkey
		marketRegistry.BaseToken = market.BaseMint
		marketRegistry.QuoteToken = market.QuoteMint
		marketRegistry.ProgramID = serum.PROGRAM_ID

		data, err := json.Marshal(marketRegistry)
		if err != nil {
			return fmt.Errorf("error marshalling abis: %w", err)
		}

		fmt.Println("Storing market!", string(data))
		_, err = f.Write(data)
		if err != nil {
			return fmt.Errorf("writing market: %w", err)
		}

		_, err = f.Write([]byte{'\n'})
		if err != nil {
			return fmt.Errorf("writing new line: %w", err)
		}
	}

	store, err := dstore.NewStore(viper.GetString("store"), "", "", true)
	if err != nil {
		return fmt.Errorf("error creating export store: %w", err)
	}
	err = store.PushLocalFile(ctx, f.Name(), newFilename)
	if err != nil {
		return fmt.Errorf("unabel to upload local file: %w", err)

	}

	return nil
}

func readKnownMarkets(ctx context.Context, marketListURL string) (map[string]*registry.Market, error) {
	out := map[string]*registry.Market{}
	err := readFile(ctx, marketListURL, func(line string) error {
		var m *registry.Market
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return fmt.Errorf("unable decode market information: %w", err)
		}
		out[m.Address.String()] = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func readFile(ctx context.Context, filepath string, f func(line string) error) error {
	reader, _, _, err := dstore.OpenObject(ctx, filepath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer reader.Close()

	bufReader := bufio.NewReader(reader)
	var line string
	for {
		line, err = bufReader.ReadString('\n')
		if err != nil {
			break
		}

		if err := f(line); err != nil {
			return fmt.Errorf("error processing line: %w", err)
		}
	}
	return nil
}
