package dgraphql

import (
	"encoding/json"
	"fmt"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/dfuse-io/dgraphql/static"
)

//go:generate rice embed-go

func GraphqlExamples() []*static.GraphqlExample {
	box := rice.MustFindBox("examples")

	return []*static.GraphqlExample{
		{
			Label:    "Get All Tokens",
			Document: graphqlDocument(box, "get_all_tokens.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"cursor": ""}`),
			},
		},
		{
			Label:    "Get A Token",
			Document: graphqlDocument(box, "get_token.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"address": "TOKEN-ADDRESS"}`),
				"mainnet": r(`{"address": "So11111111111111111111111111111111111111112"}`),
			},
		},
		{
			Label:    "Get Serum Fill",
			Document: graphqlDocument(box, "get_serum_fill.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"trader": "YOUR-ACCOUNT-HERE"}`),
				"mainnet": r(`{"trader": "5coBYaaDYd9xkMhDPDGcV2Batu51N987Um1jcrE122AY"}`),
			},
		},
		{
			Label:    "Get Serum Markets",
			Document: graphqlDocument(box, "get_serum_markets.graphql"),
		},
	}
}

func graphqlDocument(box *rice.Box, name string) static.GraphqlDocument {
	asset, err := box.String(name)
	if err != nil {
		panic(fmt.Errorf("unable to get content for graphql examples file %q: %w", name, err))
	}

	return static.GraphqlDocument(asset)
}

func oneWeekAgo() string {
	return time.Now().Add(-7 * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05Z")
}

func dateOffsetByBlock(blockCount int) string {
	return time.Now().Add(time.Duration(blockCount) * 500 * time.Millisecond).UTC().Format("2006-01-02T15:04:05Z")
}

func r(rawJSON string) json.RawMessage {
	return json.RawMessage(rawJSON)
}
