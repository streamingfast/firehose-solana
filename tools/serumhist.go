package tools

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	serumhistkeyer "github.com/dfuse-io/dfuse-solana/serumhist/keyer"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dfuse-solana/serumhist"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serumhistCmd = &cobra.Command{Use: "serumhist", Short: "Read from serum history"}

var fillCmd = &cobra.Command{
	Use:   "fill {trader-addr}",
	Short: "Read fills for a trader account",
	Long:  "Read fills for a trader account",
	Args:  cobra.ExactArgs(1),
	RunE:  readFillsE,
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
	serumhistCmd.AddCommand(fillCmd)
	serumhistCmd.AddCommand(KeyerCmd)
	KeyerCmd.AddCommand(decodeKeyerCmd)

	serumhistCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "kvStore DSN")

	fillCmd.Flags().String("market-addr", "", "Market Address")
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
	case serumhistkeyer.PrefixFillByMarketTrader:
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

func readFillsE(cmd *cobra.Command, args []string) (err error) {
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

		fills, _, err = manager.GetFillsByTraderAndMarket(cmd.Context(), trader, market)
	} else {
		fills, _, err = manager.GetFillsByTrader(cmd.Context(), trader)
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

func getKVDBAndMode() (store.KVStore, error) {
	kvdb, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return nil, fmt.Errorf("failed to setup db: %w", err)
	}
	return kvdb, nil
}
