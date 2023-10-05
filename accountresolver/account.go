package accountsresolver

import (
	"github.com/mr-tron/base58"
)

type Accounts []Account

func NewAccounts(accountHasBytes [][]byte) Accounts {
	var accounts Accounts
	for _, acc := range accountHasBytes {
		accounts = append(accounts, acc)
	}
	return accounts
}

func (a *Accounts) ToBytesArray() (out [][]byte) {
	for _, account := range *a {
		out = append(out, account)
	}
	return
}

type Account []byte

func (a Account) Base58() string {
	return base58.Encode(a)
}

func MustFromBase58(a string) Account {
	acc, err := fromBase58(a)
	if err != nil {
		panic(err)
	}
	return acc
}

func fromBase58(a string) (Account, error) {
	return base58.Decode(a)
}
