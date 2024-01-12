package fetcher

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/test-go/testify/require"
)

func Test_ToPBTransaction(t *testing.T) {

	b, err := os.ReadFile("/Users/cbillett/devel/sf/firehose-solana/block/fetcher/testdata/result_block_241179689.json")
	require.NoError(t, err)

	getBlockResult := &rpc.GetBlockResult{}
	err = json.Unmarshal(b, getBlockResult)
	require.NoError(t, err)

	_, err = toPbTransactions(getBlockResult.Transactions)
	require.NoError(t, err)

	//trxHash, err := base58.Decode("66gBszm6ybWVVykE4Svm2CvmiSmFbQi2J3Ua2FxHrYL9B1EPsTCGgjfWNVoJHSqd86iKmS8niywSZqDmqkk7uZLM")
	//require.NoError(t, err)
	//for _, tx := range confirmTransactions {
	//	if bytes.Equal(tx.Transaction.Signatures[0], trxHash) {
	//	}
	//}
}
