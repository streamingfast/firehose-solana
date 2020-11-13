package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/stretchr/testify/require"
)

func GetTransactionForAccount(ctx context.Context, rpcCient *rpc.Client, account solana.PublicKey) ([]solana.Transaction,error) {


	trx, err := rpcCient.GetConfirmedSignaturesForAddress2(ctx, account, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve transaction signatures: %w", err)
	}

	nailer := dh

	require.NoError(t, err)
	fmt.Println(string(d))

	return nil, nil
}