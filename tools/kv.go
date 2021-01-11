package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"

	serumhistkeyer "github.com/dfuse-io/dfuse-solana/serumhist/keyer"
	"github.com/dfuse-io/jsonpb"
	"github.com/dfuse-io/kvdb/store"
	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	_ "github.com/dfuse-io/kvdb/store/tikv"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var kvCmd = &cobra.Command{Use: "kv", Short: "Read from a kvStore"}

var kvPrefixCmd = &cobra.Command{Use: "prefix {prefix}", Short: "prefix read from kvStore, prefix as hex", RunE: kvPrefix, Args: cobra.ExactArgs(1)}
var kvScanCmd = &cobra.Command{Use: "scan {start} {end}", Short: "scan read from kvStore, using hex keys", RunE: kvScan, Args: cobra.ExactArgs(2)}
var kvGetCmd = &cobra.Command{Use: "get", Short: "get key from kvStore", RunE: kvGet, Args: cobra.ExactArgs(1)}

func init() {
	Cmd.AddCommand(kvCmd)
	kvCmd.AddCommand(kvPrefixCmd)
	kvCmd.AddCommand(kvScanCmd)
	kvCmd.AddCommand(kvGetCmd)

	defaultBadger := "badger://dfuse-data/kvdb/kvdb_badger.db"
	cwd, err := os.Getwd()
	if err == nil {
		defaultBadger = "badger://" + cwd + "/dfuse-data/kvdb/kvdb_badger.db"
	}

	kvCmd.PersistentFlags().String("dsn", defaultBadger, "kvStore DSN")
	kvCmd.PersistentFlags().Int("depth", 1, "Depth of decoding. 0 = top-level block, 1 = kind-specific blocks, 2 = future!")
	kvCmd.PersistentFlags().String("keyer", "", "Attempt to match key prefix and decode data. Current option 'serumhist'")
	kvScanCmd.Flags().Int("limit", 100, "limit the number of rows when doing scan")
	kvPrefixCmd.Flags().Int("limit", 100, "limit the number of rows when doing prefix")
}

func kvPrefix(cmd *cobra.Command, args []string) (err error) {
	prefix, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("error decoding prefix %q: %s", args[0], err)
	}

	// WARN: I think this `limit` doesn't work!?!
	viper.BindPFlag("limit", cmd.Flags().Lookup("limit"))
	limit := viper.GetInt("limit")

	return getPrefix(prefix, limit)
}

func kvScan(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	start, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("error decoding range start %q: %s", args[0], err)
	}
	end, err := hex.DecodeString(args[1])
	if err != nil {
		return fmt.Errorf("error decoding range end %q: %s", args[1], err)
	}

	// WARN: I think this doesn't work!?!
	viper.BindPFlag("limit", cmd.Flags().Lookup("limit"))
	limit := viper.GetInt("limit")

	it := kv.Scan(context.Background(), start, end, limit)
	for it.Next() {
		item := it.Item()
		printKVEntity(item.Key, item.Value, false, true)
	}
	if err := it.Err(); err != nil {
		return err
	}

	return nil
}

func kvGet(cmd *cobra.Command, args []string) (err error) {
	key, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("error decoding range start %q: %s", args[0], err)
	}
	return get(key)
}

func get(key []byte) error {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	val, err := kv.Get(context.Background(), key)
	if err == store.ErrNotFound {
		fmt.Printf("key %q not found\n", hex.EncodeToString(key))
		return nil
	}

	printKVEntity(key, val, false, true)

	return nil
}

func getPrefix(prefix []byte, limit int) error {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	it := kv.Prefix(context.Background(), prefix, limit)
	for it.Next() {
		item := it.Item()
		printKVEntity(item.Key, item.Value, false, true)
	}
	if err := it.Err(); err != nil {
		return err
	}

	return nil
}

func printKVEntity(key, val []byte, asHex bool, indented bool) (err error) {
	if asHex {
		fmt.Println(hex.EncodeToString(key), hex.EncodeToString(val))
		return nil
	}

	row := map[string]interface{}{
		"key": hex.EncodeToString(key),
	}

	row["data"], err = decodeKey(key[0], val)

	cnt, err := json.Marshal(row)
	if err != nil {
		return err
	}
	fmt.Println(string(cnt))
	return nil
}

func decodeKey(keyPrefix byte, val []byte) (out interface{}, err error) {
	pbmarsh := jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: true,
		OrigName:     true,
	}

	switch viper.GetString("keyer") {
	case "serumhist":
		switch keyPrefix {
		case serumhistkeyer.PrefixFillData:
			protoMessage := &pbserumhist.Fill{}
			out, err = decodePayload(pbmarsh, protoMessage, val)
		case serumhistkeyer.PrefixCheckpoint:
			protoMessage := &pbserumhist.Checkpoint{}
			out, err = decodePayload(pbmarsh, protoMessage, val)
		}
	}
	if out == nil {
		out = hex.EncodeToString(val)
	}
	return
}

func decodePayload(marshaler jsonpb.Marshaler, obj proto.Message, bytes []byte) (out json.RawMessage, err error) {
	err = proto.Unmarshal(bytes, obj)
	if err != nil {
		return nil, fmt.Errorf("proto unmarshal: %s", err)
	}

	cnt, err := marshaler.MarshalToString(obj)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %s", err)
	}

	return json.RawMessage(cnt), nil
}
