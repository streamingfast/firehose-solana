package meta

import (
	"testing"

	"github.com/dfuse-io/solana-go"
	"github.com/stretchr/testify/require"
)

func TestTokenMeta_GetTokeInfo(t *testing.T) {

	tm, err := NewTokenMeta()
	require.NoError(t, err)

	tokenInfo := tm.GetTokeInfo(solana.MustPublicKeyFromBase58("MSRMcoVyrFxnSgo5uXwone5SKcGhT1KEJMFEkMEWf9L"))
	require.Equal(t, "MegaSerum", tokenInfo.Name)
	require.Equal(t, "MSRM", tokenInfo.Symbol)

}
