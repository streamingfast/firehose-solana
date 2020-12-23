package tools

import (
	"encoding/json"
	"fmt"

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

func init() {
	Cmd.AddCommand(serumhistCmd)
	serumhistCmd.AddCommand(fillCmd)

	serumhistCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "kvStore DSN")

	fillCmd.Flags().String("market-addr", "", "Market Address")
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

		fills, err = manager.GetFillsByTraderAndMarket(cmd.Context(), trader, market)
	} else {
		fills, err = manager.GetFillsByTrader(cmd.Context(), trader)
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
