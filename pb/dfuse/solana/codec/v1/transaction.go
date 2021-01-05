package pbcodec

import (
	"fmt"

	"github.com/dfuse-io/solana-go"
)

func (t *Transaction) IsSigner(account string) bool {
	for idx, acc := range t.AccountKeys {
		if acc == account {
			return idx < int(t.Header.NumRequiredSignatures)
		}
	}
	return false
}

func (t *Transaction) IsWritable(account string) bool {
	index := 0
	found := false
	for idx, acc := range t.AccountKeys {
		if acc == account {
			found = true
			index = idx
		}
	}
	if !found {
		return false
	}
	h := t.Header
	return (index < int(h.NumRequiredSignatures-h.NumReadonlySignedAccounts)) ||
		((index >= int(h.NumRequiredSignatures)) && (index < len(t.AccountKeys)-int(h.NumReadonlyUnsignedAccounts)))
}

func (t *Transaction) ResolveProgramIDIndex(programIDIndex uint8) (solana.PublicKey, error) {
	if int(programIDIndex) < len(t.AccountKeys) {
		return solana.PublicKeyFromBase58(t.AccountKeys[programIDIndex])
	}
	return solana.PublicKey{}, fmt.Errorf("programID index not found %d", programIDIndex)
}

func (t *Transaction) AccountMetaList() (out []*solana.AccountMeta, err error) {
	for _, acc := range t.AccountKeys {
		account, err := solana.PublicKeyFromBase58(acc)
		if err != nil {
			return nil, fmt.Errorf("account meta list: account to pub key: %w", err)
		}
		out = append(out, &solana.AccountMeta{
			PublicKey:  account,
			IsSigner:   t.IsSigner(acc),
			IsWritable: t.IsWritable(acc),
		})
	}
	return out, nil
}
