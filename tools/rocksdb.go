package tools

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
	"github.com/tecbot/gorocksdb"
)

var rocksCmd = &cobra.Command{Use: "rocks", Short: "Read from rocksdb"}
var rocksGetCmd = &cobra.Command{Use: "get", Short: "get key from kvStore", RunE: rocksGet, Args: cobra.ExactArgs(1)}

func init() {
	Cmd.AddCommand(rocksCmd)
	kvCmd.AddCommand(rocksGetCmd)

	kvCmd.PersistentFlags().String("path", "", "path to rocksdb")
}

func rocksGet(cmd *cobra.Command, args []string) (err error) {

	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	db, err := gorocksdb.OpenDb(opts, viper.GetString("dsn"))

	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	return
}
