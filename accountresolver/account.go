package solana_accounts_resolver

import (
	"errors"

	"github.com/mr-tron/base58"
)

type Account []byte

var AccountNotFound = errors.New("account not found")

func (a Account) base58() string {
	return base58.Encode(a)
}
