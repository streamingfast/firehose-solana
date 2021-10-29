package pbcodec

import (
	"bytes"
	"fmt"

	"github.com/streamingfast/solana-go"
)

func (t *Transaction) IsSigner(account []byte) bool {
	for idx, acc := range t.AccountKeys {
		if bytes.Equal(acc, account) {
			return idx < int(t.Header.NumRequiredSignatures)
		}
	}
	return false
}

func (t *Transaction) IsWritable(account []byte) bool {
	index := 0
	found := false
	for idx, acc := range t.AccountKeys {
		if bytes.Equal(acc, account) {
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

func (t *Transaction) ResolveProgramIDIndex(programIDIndex uint8) (out solana.PublicKey, err error) {
	if int(programIDIndex) < len(t.AccountKeys) {
		return solana.PublicKeyFromBytes(t.AccountKeys[programIDIndex]), nil
	}

	return out, fmt.Errorf("programID index not found %d", programIDIndex)
}

func (t *Transaction) AccountMetaList() (out []*solana.AccountMeta, err error) {
	out = make([]*solana.AccountMeta, len(t.AccountKeys))
	for i, acc := range t.AccountKeys {
		out[i] = &solana.AccountMeta{
			PublicKey:  solana.PublicKeyFromBytes(acc),
			IsSigner:   t.IsSigner(acc),
			IsWritable: t.IsWritable(acc),
		}
	}

	return out, nil
}

func (t *Transaction) InstructionAccountMetaList(i *Instruction) (out []*solana.AccountMeta) {
	out = make([]*solana.AccountMeta, len(i.AccountKeys))
	for i, acc := range i.AccountKeys {
		out[i] = &solana.AccountMeta{
			PublicKey:  solana.PublicKeyFromBytes(acc),
			IsSigner:   t.IsSigner(acc),
			IsWritable: t.IsWritable(acc),
		}
	}

	return out
}
