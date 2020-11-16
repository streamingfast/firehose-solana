package transaction

import (
	"context"
	"fmt"

	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"go.uber.org/zap"
)

type TransactionHandlerFunc func(trx *rpc.TransactionWithMeta)

func GetTransactionForAccount(ctx context.Context, rpcCient *rpc.Client, account solana.PublicKey, handler TransactionHandlerFunc) (err error) {
	signatures, err := rpcCient.GetConfirmedSignaturesForAddress2(ctx, account, nil)
	if err != nil {
		return fmt.Errorf("unable to retrieve transaction signatures: %w", err)
	}

	zlog.Debug("retrieved trx signatures for account",
		zap.Stringer("account", account),
		zap.Int("singatures_count", len(signatures)),
	)

	for _, sig := range signatures {
		if traceEnabled {
			zlog.Debug("retrieving trx",
				zap.String("signature", sig.Signature),
			)
		}
		trx, err := rpcCient.GetConfirmedTransaction(ctx, sig.Signature)
		if err != nil {
			return fmt.Errorf("unable to retrieve transaction with signature %q, : %w", sig.Signature, err)
		}

		if traceEnabled {
			zlog.Debug("retrieved trx",
				zap.String("signature", sig.Signature),
				zap.Int("instruction_count", len(trx.Transaction.Message.Instructions)),
				zap.Int("account_key_count", len(trx.Transaction.Message.AccountKeys)),
			)
		}

		handler(&trx)
	}
	return nil
}
