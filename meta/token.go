package meta

import (
	"encoding/json"
	"fmt"

	"github.com/dfuse-io/solana-go"

	rice "github.com/GeertJohan/go.rice"
)

//go:generate rice embed-go

type TokenInfo struct {
	Symbol  string `json:"tokenSymbol"`
	Address string `json:"mintAddress"`
	Name    string `json:"tokenName"`
	IconURL string `json:"icon,omitempty"`
}

type TokenMeta struct {
	tokens map[string]*TokenInfo
}

func NewTokenMeta() (*TokenMeta, error) {
	box := rice.MustFindBox("statics")

	tokenMeta := &TokenMeta{
		tokens: map[string]*TokenInfo{},
	}

	var tokens []*TokenInfo

	data, err := box.Bytes("tokens.json")
	if err != nil {
		return nil, fmt.Errorf("new token meta: box bytes: %w", err)
	}

	err = json.Unmarshal(data, &tokens)
	if err != nil {
		return nil, fmt.Errorf("new token meta: json unmarshall: %w", err)
	}

	for _, t := range tokens {
		tokenMeta.tokens[t.Address] = t
	}

	return tokenMeta, nil
}

func (m TokenMeta) GetTokeInfo(address solana.PublicKey) *TokenInfo {
	return m.tokens[address.String()]
}
