package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"go.uber.org/zap"
)

func makeDirs(directories []string) error {
	for _, directory := range directories {
		err := os.MkdirAll(directory, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %q: %w", directory, err)
		}
	}

	return nil
}

func MustReplaceDataDir(dataDir, in string) string {
	d, err := filepath.Abs(dataDir)
	if err != nil {
		panic(fmt.Errorf("file path abs: %w", err))
	}

	in = strings.Replace(in, "{data-dir}", d, -1)
	return in
}

func mkdirStorePathIfLocal(storeURL string) (err error) {
	zlog.Debug("creating directory and its parent(s)", zap.String("directory", storeURL))
	if dirs := getDirsToMake(storeURL); len(dirs) > 0 {
		err = makeDirs(dirs)
	}
	return
}

func getDirsToMake(storeURL string) []string {
	parts := strings.Split(storeURL, "://")
	if len(parts) > 1 {
		if parts[0] != "file" {
			// Not a local store, nothing to do
			return nil
		}
		storeURL = parts[1]
	}

	// Some of the store URL are actually a file directly, let's try our best to cope for that case
	filename := filepath.Base(storeURL)
	if strings.Contains(filename, ".") {
		storeURL = filepath.Dir(storeURL)
	}

	// If we reach here, it's a local store path
	return []string{storeURL}
}

var commonStoresCreated bool

func getCommonStoresURLs(dataDir string) (mergedBlocksStoreURL, oneBlockStoreURL, forkedBlocksStoreURL string, err error) {
	mergedBlocksStoreURL = MustReplaceDataDir(dataDir, viper.GetString("common-merged-blocks-store-url"))
	oneBlockStoreURL = MustReplaceDataDir(dataDir, viper.GetString("common-one-block-store-url"))
	forkedBlocksStoreURL = MustReplaceDataDir(dataDir, viper.GetString("common-forked-blocks-store-url"))

	if commonStoresCreated {
		return
	}

	if err = mkdirStorePathIfLocal(forkedBlocksStoreURL); err != nil {
		return
	}
	if err = mkdirStorePathIfLocal(oneBlockStoreURL); err != nil {
		return
	}
	if err = mkdirStorePathIfLocal(mergedBlocksStoreURL); err != nil {
		return
	}
	commonStoresCreated = true
	return
}
