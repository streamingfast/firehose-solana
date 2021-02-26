package tools

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	serumhistkeyer "github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	serumhistreader "github.com/dfuse-io/dfuse-solana/serumhist/reader"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dfuse-io/solana-go"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serumhistCmd = &cobra.Command{Use: "serumhist", Short: "Read from serum history"}

var marketsCmd = &cobra.Command{
	Use:   "markets",
	Short: "list markets",
	RunE:  readMarketsE,
}

var tradersCmd = &cobra.Command{
	Use:   "traders",
	Short: "list traders",
	RunE:  readTradersE,
}

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

var ordersCmd = &cobra.Command{
	Use:   "orders",
	Short: "Read orders",
}

var getOrdersCmd = &cobra.Command{
	Use:   "get {market-addr} {order-num}",
	Short: "Read an order",
	RunE:  readGetOrdersE,
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
	serumhistCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "kvStore DSN")

	serumhistCmd.AddCommand(marketsCmd)
	serumhistCmd.AddCommand(tradersCmd)

	serumhistCmd.AddCommand(fillsCmd)
	fillsCmd.AddCommand(traderFillsCmd)
	fillsCmd.AddCommand(marketFillsCmd)
	fillsCmd.Flags().Int("limit", 100, "Number of fills to retrieve")
	traderFillsCmd.Flags().String("market-addr", "", "Market Address")

	serumhistCmd.AddCommand(ordersCmd)
	ordersCmd.AddCommand(getOrdersCmd)

	serumhistCmd.AddCommand(KeyerCmd)
	KeyerCmd.AddCommand(decodeKeyerCmd)

	serumhistCmd.AddCommand(checkpointCmd)

}

func decoderKeyerE(cmd *cobra.Command, args []string) (err error) {
	keyStr := args[0]
	key, err := hex.DecodeString(keyStr)
	if err != nil {
		return fmt.Errorf("unable to decode key: %w", err)
	}
	switch key[0] {
	case serumhistkeyer.PrefixFill:
		market, slotNum, trxIdx, instIdx, orderSeqNum := serumhistkeyer.DecodeFill(key)
		printDecodedKey("Fill Key", market.String(), slotNum, trxIdx, instIdx, orderSeqNum)
	case serumhistkeyer.PrefixFillByTrader:
		trader, market, slotNum, trxIdx, instIdx, orderSeqNum := serumhistkeyer.DecodeFillByTrader(key)
		printDecodedKey("Fill by trader", market.String(), slotNum, trxIdx, instIdx, orderSeqNum)
		fmt.Println("Trader: ", trader.String())
	case serumhistkeyer.PrefixFillByTraderMarket:
		trader, market, slotNum, trxIdx, instIdx, orderSeqNum := serumhistkeyer.DecodeFillByTraderMarket(key)
		printDecodedKey("Fill by trader market trader", market.String(), slotNum, trxIdx, instIdx, orderSeqNum)
		fmt.Println("Trader: ", trader.String())
	case serumhistkeyer.PrefixOrder:
		event := ""
		eventPrefix, market, slotNum, trxIdx, instIdx, orderSeqNum := serumhistkeyer.DecodeOrder(key)
		switch eventPrefix {
		case serumhistkeyer.OrderEventTypeNew:
			event = "New Order"
		case serumhistkeyer.OrderEventTypeFill:
			event = "Order Filled"
		case serumhistkeyer.OrderEventTypeExecuted:
			event = "Order Executed"
		case serumhistkeyer.OrderEventTypeCancel:
			event = "Order Canceled"
		case serumhistkeyer.OrderEventTypeClose:
			event = "Order closed"
		}
		printDecodedKey(fmt.Sprintf("Order key for event: %q", event), market.String(), slotNum, trxIdx, instIdx, orderSeqNum)
	case serumhistkeyer.PrefixOrderByMarket:
		market, orderSeqNum := serumhistkeyer.DecodeOrderByMarket(key)
		fmt.Println("Order by market")
		fmt.Println("Market:", market)
		fmt.Println("Order Seq Num:", orderSeqNum)
	case serumhistkeyer.PrefixOrderByTrader:
		trader, market, slotNum, trxIdx, instIdx, orderSeqNum := serumhistkeyer.DecodeOrderByTrader(key)
		printDecodedKey("Order by trader", market.String(), slotNum, trxIdx, instIdx, orderSeqNum)
		fmt.Println("Trader: ", trader.String())
	case serumhistkeyer.PrefixOrderByTraderMarket:
		trader, market, slotNum, trxIdx, instIdx, orderSeqNum := serumhistkeyer.DecodeOrderByTraderMarket(key)
		printDecodedKey("Order by trader market", market.String(), slotNum, trxIdx, instIdx, orderSeqNum)
		fmt.Println("Trader: ", trader.String())
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

func readGetOrdersE(cmd *cobra.Command, args []string) error {
	kvdb, err := getKVDBAndMode()
	if err != nil {
		return err
	}
	reader := serumhistreader.New(kvdb)
	marketAddr := args[0]
	market, err := solana.PublicKeyFromBase58(marketAddr)
	if err != nil {
		return fmt.Errorf("unable to create market public key: %w", err)
	}

	orderSeqNum, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse order num %q: %w", args[1], err)
	}
	order, err := reader.GetOrder(cmd.Context(), market, orderSeqNum)
	if err != nil {
		return fmt.Errorf("unable to get order for market %q & num %d: %w", market.String(), orderSeqNum, err)
	}

	cnt, err := json.MarshalIndent(order, "", " ")
	if err != nil {
		return fmt.Errorf("unable to marshall: %w", err)
	}
	fmt.Println(string(cnt))
	return nil
}

func readMarketsE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := getKVDBAndMode()
	if err != nil {
		return err
	}
	start := []byte{serumhistkeyer.PrefixFill}
	stop := []byte{serumhistkeyer.PrefixFill + 1}
	fmt.Println("Markets:")
	for {
		iter := kvdb.Scan(cmd.Context(), start, stop, 0)
		if iter.Next() {
			market, _, _, _, _ := serumhistkeyer.DecodeFill(iter.Item().Key)
			fmt.Println(market.String())
			key := make([]byte, 1+32)
			key[0] = serumhistkeyer.PrefixFill
			copy(key[1:], market[:])
			start = store.Key(key).PrefixNext()
		} else {
			return
		}
	}
}
func readTradersE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := getKVDBAndMode()
	if err != nil {
		return err
	}
	start := []byte{serumhistkeyer.PrefixFillByTrader}
	stop := []byte{serumhistkeyer.PrefixFillByTrader + 1}
	fmt.Println("Traders:")
	for {
		iter := kvdb.Scan(cmd.Context(), start, stop, 0)
		if iter.Next() {
			trader, _, _, _, _, _ := serumhistkeyer.DecodeFillByTrader(iter.Item().Key)
			fmt.Println(trader.String())
			key := make([]byte, 1+32)
			key[0] = serumhistkeyer.PrefixFillByTrader
			copy(key[1:], trader[:])
			start = store.Key(key).PrefixNext()
		} else {
			return
		}
	}
}

func readTraderFillsE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := getKVDBAndMode()
	if err != nil {
		return err
	}

	var fills []*pbserumhist.Fill
	reader := serumhistreader.New(kvdb)

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
		fills, _, err = reader.GetFillsByTraderAndMarket(cmd.Context(), trader, market, viper.GetInt("limit"))
	} else {
		fmt.Println("getting fills for trader", trader.String())
		fills, _, err = reader.GetFillsByTrader(cmd.Context(), trader, viper.GetInt("limit"))
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
	reader := serumhistreader.New(kvdb)

	marketAddr := args[0]
	market, err := solana.PublicKeyFromBase58(marketAddr)
	if err != nil {
		return fmt.Errorf("unable to create public key: %w", err)
	}

	// fmt.Println("getting fills for market", market.String())
	fills, _, err = reader.GetFillsByMarket(cmd.Context(), market, viper.GetInt("limit"))

	cnt, err := json.Marshal(fills)
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

func printDecodedKey(title, market string, slotNum, trxIdx, instIdx, orderSeqNum uint64) {
	fmt.Println(title)
	fmt.Println("Market:", market)
	fmt.Println("Slot:", slotNum)
	fmt.Println("Trx idx:", trxIdx)
	fmt.Println("Inst idx:", instIdx)
	fmt.Println("Order Seq Num:", orderSeqNum)
}
