package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/dfuse-io/dfuse-solana/registry"
	"github.com/streamingfast/dstore"
	"github.com/dfuse-io/solana-go/programs/serum"
	solrpc "github.com/dfuse-io/solana-go/rpc"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{Use: "registry", Short: "Registry "}
var listRegistryCmd = &cobra.Command{Use: "list", Short: "List entities"}
var fetchRegistryCmd = &cobra.Command{Use: "fetch", Short: "Fetch entities"}
var marketsFetchCmd = &cobra.Command{
	Use:   "markets {old-markets.jsonl} {new-markets.jsonl}",
	Short: "Retrieve Serum Markets",
	Long:  "Retrieve Serum Markets",
	Args:  cobra.ExactArgs(2),
	RunE:  fetchMarketsE,
}

var tokensFetchCmd = &cobra.Command{
	Use:   "tokens {old-tokens.jsonl} {new-tokens.jsonl}",
	Short: "Retrieve Solana Tokens",
	Long:  "Retrieve Solana Tokens",
	Args:  cobra.ExactArgs(2),
	RunE:  fetchTokensE,
}

func init() {
	Cmd.AddCommand(registryCmd)
	registryCmd.AddCommand(fetchRegistryCmd)
	fetchRegistryCmd.AddCommand(tokensFetchCmd)
	fetchRegistryCmd.AddCommand(marketsFetchCmd)

	registryCmd.PersistentFlags().String("rpc", "https://solana-api.projectserum.com", "RPC URL")
	marketsFetchCmd.PersistentFlags().String("store", "gs://staging.dfuseio-global.appspot.com/sol-markets", "Market store")
	tokensFetchCmd.PersistentFlags().String("store", "gs://staging.dfuseio-global.appspot.com/sol-tokens", "Token store")
}

func fetchMarketsE(cmd *cobra.Command, args []string) (err error) {
	// todo this should be an argument
	oldFilepath := fmt.Sprintf("%s/%s", viper.GetString("store"), args[0])
	newFilename := args[1]

	ctx := cmd.Context()
	client := solrpc.NewClient(viper.GetString("rpc"))
	fmt.Printf("Retrieving outdated markets: %s\n", oldFilepath)
	oldMarkets, err := registry.ReadKnownMarkets(ctx, oldFilepath)
	if err != nil {
		return fmt.Errorf("unable to retrieve known markets: %w", err)
	}
	fmt.Printf("Loaded %d known markets\n", len(oldMarkets))

	fmt.Printf("Retrieving on-chain markets for program: %s\n", serum.DEXProgramIDV2.String())
	accounts, err := client.GetProgramAccounts(
		ctx,
		serum.DEXProgramIDV2,
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
	fmt.Printf("Found %d on chain markets for program %s\n", len(accounts), serum.DEXProgramIDV2.String())

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
		marketRegistry.ProgramID = serum.DEXProgramIDV2
		marketRegistry.BaseToken = market.BaseMint
		marketRegistry.QuoteToken = market.QuoteMint
		marketRegistry.BaseLotSize = uint64(market.BaseLotSize)
		marketRegistry.QuoteLotSize = uint64(market.QuoteLotSize)
		marketRegistry.RequestQueue = market.RequestQueue
		marketRegistry.EventQueue = market.EventQueue

		data, err := json.Marshal(marketRegistry)
		if err != nil {
			return fmt.Errorf("error marshalling market: %w", err)
		}

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

func fetchTokensE(cmd *cobra.Command, args []string) (err error) {
	// todo this should be an argument
	oldFilepath := fmt.Sprintf("%s/%s", viper.GetString("store"), args[0])
	newFilename := args[1]

	ctx := cmd.Context()
	client := solrpc.NewClient(viper.GetString("rpc"))
	fmt.Printf("Retrieving outdated tokens: %s\n", oldFilepath)
	oldTokens, err := registry.ReadKnownTokens(ctx, oldFilepath)
	if err != nil {
		return fmt.Errorf("unable to retrieve known tokens: %w", err)
	}

	tokens := make([]*registry.Token, len(oldTokens))
	i := 0
	for _, t := range oldTokens {
		if t.Meta != nil {
			t.Verified = true
		}
		tokens[i] = t
		i++
	}

	out, err := registry.SyncKnownTokens(client, tokens)
	if err != nil {
		return fmt.Errorf("unable to sync tokens: %w", err)
	}
	fmt.Printf("Synced %d tokens\n", len(out))

	f, err := os.Create(filepath.Join("/tmp", "out.jsonl"))
	if err != nil {
		return fmt.Errorf("unable to open local file write: %w", err)
	}
	defer f.Close()

	for _, t := range out {
		data, err := json.Marshal(t)
		if err != nil {
			return fmt.Errorf("error marshalling tokens: %w", err)
		}

		_, err = f.Write(data)
		if err != nil {
			return fmt.Errorf("writing token: %w", err)
		}

		_, err = f.Write([]byte{'\n'})
		if err != nil {
			return fmt.Errorf("writing new line: %w", err)
		}
	}

	//fmt.Printf("Retrieving on-chain tokens: %s\n", token.TOKEN_PROGRAM_ID.String())
	//accounts, err := client.GetProgramAccounts(
	//	ctx,
	//	token.TOKEN_PROGRAM_ID,
	//	&solrpc.GetProgramAccountsOpts{
	//		Filters: []solrpc.RPCFilter{
	//			{
	//				DataSize: token.MINT_SIZE,
	//			},
	//		},
	//	},
	//)
	//if err != nil {
	//	return fmt.Errorf("unable to retrive tokens from chain: %w", err)
	//}
	//fmt.Printf("Found %d on chain tokens\n", len(accounts))
	//
	//f, err := os.Create(filepath.Join("/tmp", "out.jsonl"))
	//if err != nil {
	//	return fmt.Errorf("unable to open local file write: %w", err)
	//}
	//defer f.Close()
	//
	//for _, acc := range tokens {
	//
	//	mint := &token.Mint{}
	//	err := mint.Decode(acc.Account.Data)
	//	if err != nil {
	//		fmt.Printf("Skipping account %s unable to decode market\n", acc.Pubkey.String())
	//		continue
	//	}
	//
	//	tokenRegistry := &registry.Token{}
	//	if t, found := oldTokens[acc.Pubkey.String()]; found {
	//		tokenRegistry = t
	//	}
	//
	//	tokenRegistry.MintAuthorityOption = mint.MintAuthorityOption
	//	tokenRegistry.MintAuthority = mint.MintAuthority
	//	tokenRegistry.Supply = uint64(mint.Supply)
	//	tokenRegistry.Decimals = mint.Decimals
	//	tokenRegistry.IsInitialized = mint.IsInitialized
	//	tokenRegistry.FreezeAuthorityOption = mint.FreezeAuthorityOption
	//	tokenRegistry.FreezeAuthority = mint.FreezeAuthority
	//
	//	data, err := json.Marshal(tokenRegistry)
	//	if err != nil {
	//		return fmt.Errorf("error marshalling tokens: %w", err)
	//	}
	//
	//	_, err = f.Write(data)
	//	if err != nil {
	//		return fmt.Errorf("writing token: %w", err)
	//	}
	//
	//	_, err = f.Write([]byte{'\n'})
	//	if err != nil {
	//		return fmt.Errorf("writing new line: %w", err)
	//	}
	//}

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
