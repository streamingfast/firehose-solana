package tools

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist"
	serumhistkeyer "github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serumhistCmd = &cobra.Command{Use: "serumhist", Short: "Read from serum history"}

// dfusesol tools serumhist fills market {}
var fillsCmd = &cobra.Command{
	Use:   "fills",
	Short: "Read fills",
}

var traderFillsCmd = &cobra.Command{
	Use:   "trader {trader-addr}",
	Short: "Read fills by trader",
	RunE:  readTraderFillsE,
}

var marketFillsCmd = &cobra.Command{
	Use:   "market {market-addr}",
	Short: "Read fills by market",
	RunE:  readMarketFillsE,
}

var checkpointCmd = &cobra.Command{
	Use:   "checkpoint",
	Short: "Get checkpoint",
	Long:  "Get checkpoint",
	RunE:  readCheckpointE,
}

var KeyerCmd = &cobra.Command{Use: "keyer", Short: "Serum history keyer helpers"}
var decodeKeyerCmd = &cobra.Command{
	Use:   "decode {key}",
	Short: "Decodes a serum key",
	Long:  "Decodes a serum key",
	Args:  cobra.ExactArgs(1),
	RunE:  decoderKeyerE,
}

func init() {
	Cmd.AddCommand(serumhistCmd)
	serumhistCmd.AddCommand(fillsCmd)
	fillsCmd.AddCommand(traderFillsCmd)
	fillsCmd.AddCommand(marketFillsCmd)

	serumhistCmd.AddCommand(KeyerCmd)
	serumhistCmd.AddCommand(checkpointCmd)
	KeyerCmd.AddCommand(decodeKeyerCmd)

	serumhistCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "kvStore DSN")

	traderFillsCmd.Flags().String("market-addr", "", "Market Address")
	fillsCmd.Flags().Int("limit", 100, "Number of fills to retrieve")
}

func decoderKeyerE(cmd *cobra.Command, args []string) (err error) {
	keyStr := args[0]
	key, err := hex.DecodeString(keyStr)
	if err != nil {
		return fmt.Errorf("unable to decode key: %w", err)
	}
	switch key[0] {
	case serumhistkeyer.PrefixFillByTrader:
		trader, market, slotNum, trxIdx, instIdx, orderSeqNum := serumhistkeyer.DecodeFillByTrader(key)
		fmt.Println("Fill By Trader Key:")
		fmt.Println("Trader:", trader.String())
		fmt.Println("Marker:", market.String())
		fmt.Println("Slot Num:", slotNum)
		fmt.Println("Trx idx:", trxIdx)
		fmt.Println("Inst idx:", instIdx)
		fmt.Println("Order Seq Num:", orderSeqNum)
	case serumhistkeyer.PrefixFillByTraderMarket:
		trader, market, slotNum, trxIdx, instIdx, orderSeqNum := serumhistkeyer.DecodeFillByMarketTrader(key)
		fmt.Println("Fill By Trader Key:")
		fmt.Println("Trader:", trader.String())
		fmt.Println("Marker:", market.String())
		fmt.Println("Slot Num:", slotNum)
		fmt.Println("Trx idx:", trxIdx)
		fmt.Println("Inst idx:", instIdx)
		fmt.Println("Order Seq Num:", orderSeqNum)
	case serumhistkeyer.PrefixTradingAccount:
		traderAccount := serumhistkeyer.DecodeTradingAccount(key)
		fmt.Println("Trading Account Key :")
		fmt.Println("Marker:", traderAccount.String())
	}
	return nil
}

func readCheckpointE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := getKVDBAndMode()
	if err != nil {
		return err
	}

	key := serumhistkeyer.EncodeCheckpoint()

	val, err := kvdb.Get(cmd.Context(), key)
	if err == store.ErrNotFound {
		fmt.Println("No checkpoint found")
		return nil
	} else if err != nil {
		return fmt.Errorf("error reading checkpoint: %w", err)

	}

	// Decode val as `pbaccounthist.ShardCheckpoint`
	out := &pbserumhist.Checkpoint{}
	if err := proto.Unmarshal(val, out); err != nil {
		return fmt.Errorf("error marhsalling checkpoint: %w", err)
	}

	fmt.Println("Checkpoint found:")
	fmt.Println("LastWrittenSlotNum: ", out.LastWrittenSlotNum)
	fmt.Println("LastWrittenSlotId: ", out.LastWrittenSlotId)
	return nil
}

func readTraderFillsE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := getKVDBAndMode()
	if err != nil {
		return err
	}

	var fills []*pbserumhist.Fill
	manager := serumhist.NewManager(kvdb)

	traderAddr := args[0]
	trader, err := solana.PublicKeyFromBase58(traderAddr)
	if err != nil {
		return fmt.Errorf("unable to create trader public key: %w", err)
	}

	marketStr := viper.GetString("market-addr")
	if marketStr != "" {
		market, err := solana.PublicKeyFromBase58(traderAddr)
		if err != nil {
			return fmt.Errorf("unable to create public key: %w", err)
		}

		fmt.Printf("getting fills for trader %s and market %s\n", trader.String(), market.String())
		fills, _, err = manager.GetFillsByTraderAndMarket(cmd.Context(), trader, market, viper.GetInt("limit"))
	} else {
		fmt.Println("getting fills for trader", trader.String())
		fills, _, err = manager.GetFillsByTrader(cmd.Context(), trader, viper.GetInt("limit"))
	}

	if err != nil {
		return err
	}

	cnt, err := json.MarshalIndent(fills, "", " ")
	if err != nil {
		return fmt.Errorf("unable to marshall: %w", err)
	}
	fmt.Println(string(cnt))
	return nil
}

func readMarketFillsE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := getKVDBAndMode()
	if err != nil {
		return err
	}

	var fills []*pbserumhist.Fill
	manager := serumhist.NewManager(kvdb)

	marketAddr := args[0]
	market, err := solana.PublicKeyFromBase58(marketAddr)
	if err != nil {
		return fmt.Errorf("unable to create public key: %w", err)
	}

	fmt.Println("getting fills for market", market.String())
	fills, _, err = manager.GetFillsByMarket(cmd.Context(), market, viper.GetInt("limit"))

	cnt, err := json.MarshalIndent(fills, "", " ")
	if err != nil {
		return fmt.Errorf("unable to marshall: %w", err)
	}
	fmt.Println(string(cnt))
	return nil
}

func getKVDBAndMode() (store.KVStore, error) {
	kvdb, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return nil, fmt.Errorf("failed to setup db: %w", err)
	}
	return kvdb, nil
}
