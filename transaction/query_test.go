package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetTransaction(t *testing.T) {
	rpc := rpc.NewClient("http://api.mainnet-beta.solana.com:80/rpc")
	account := solana.MustPublicKeyFromBase58("CG1XSWuXo2rw2SuHTRc54nihKvLKh4wMYi7oF3487LYt")
	transactions, err := GetTransactionForAccount(context.Background(),rpc,  account)
	require.NoError(t, err)

	d, err := json.MarshalIndent(transactions, "", " ")
	require.NoError(t, err)
	fmt.Println(string(d))
}