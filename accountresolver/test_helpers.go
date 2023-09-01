package accountsresolver

import (
	"github.com/mr-tron/base58"
	"testing"
)

func fromBase58Strings(t *testing.T, vals ...string) (accounts Accounts) {
	t.Helper()
	for _, s := range vals {
		accounts = append(accounts, accountFromBase58(t, s))
	}
	return accounts
}

func accountFromBase58(t *testing.T, account string) Account {
	t.Helper()
	data, err := base58.Decode(account)
	if err != nil {
		panic(err)
	}
	return data
}
