package tools

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path"
	"strings"
	"testing"
)

func TestNewSnapshotReader(t *testing.T) {

	reader, err := NewSnapshotReader("gs://mainnet-beta-ledger-us-ny5")
	require.NoError(t, err)

	filepath := path.Join(strings.TrimLeft(reader.bucket.Path, "/"), "87695515", "bounds.txt")
	filereader, err := reader.client.Bucket(reader.bucket.Host).Object(filepath).NewReader(context.Background())
	require.NoError(t, err)

	data, err := ioutil.ReadAll(filereader)
	require.NoError(t, err)
	fmt.Println(string(data))
	mactches := boundsRegEx.FindStringSubmatch(string(data))
	fmt.Printf("%#v\n", matches)
	if len(matches)
		fmt.Printf("%#v\n", boundsRegEx.SubexpNames())

}
