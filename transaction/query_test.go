package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

func TestGetTransaction(t *testing.T) {
	t.Skip()
	rpcClient := rpc.NewClient("http://api.mainnet-beta.solana.com:80/rpc")
	account := solana.MustPublicKeyFromBase58("CG1XSWuXo2rw2SuHTRc54nihKvLKh4wMYi7oF3487LYt")
	err := GetTransactionForAccount(context.Background(), rpcClient, account, func(trx *rpc.TransactionWithMeta) {
		d, err := json.MarshalIndent(trx, "", " ")
		require.NoError(t, err)
		fmt.Println(string(d))
	})
	require.NoError(t, err)
}
